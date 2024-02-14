package drift

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/yourorg/gitops-drift-detector/internal/alert"
	"github.com/yourorg/gitops-drift-detector/internal/audit"
	"github.com/yourorg/gitops-drift-detector/internal/cluster"
	"github.com/yourorg/gitops-drift-detector/internal/git"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type DriftType string

const (
	DriftModified   DriftType = "modified"
	DriftMissing    DriftType = "missing"
	DriftUnexpected DriftType = "unexpected"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

type DriftEvent struct {
	ID           string          `json:"id"`
	ClusterID    string          `json:"cluster_id"`
	ResourceKind string          `json:"resource_kind"`
	ResourceName string          `json:"resource_name"`
	Namespace    string          `json:"namespace"`
	Severity     Severity        `json:"severity"`
	DriftType    DriftType       `json:"drift_type"`
	DesiredState json.RawMessage `json:"desired_state,omitempty"`
	LiveState    json.RawMessage `json:"live_state,omitempty"`
	Diff         json.RawMessage `json:"diff,omitempty"`
	DetectedAt   time.Time       `json:"detected_at"`
}

type Engine struct {
	clusters  *cluster.Manager
	git       *git.Poller
	audit     *audit.Logger
	alerter   *alert.Alerter
	log       *zap.Logger

	// latest snapshot per cluster, used by the API
	mu     sync.RWMutex
	latest map[string][]DriftEvent
}

func NewEngine(
	clusters *cluster.Manager,
	git *git.Poller,
	audit *audit.Logger,
	alerter *alert.Alerter,
	log *zap.Logger,
) *Engine {
	return &Engine{
		clusters: clusters,
		git:      git,
		audit:    audit,
		alerter:  alerter,
		log:      log,
		latest:   make(map[string][]DriftEvent),
	}
}

func (e *Engine) Run(ctx context.Context) {
	// git poller runs independently; we just react to its cache
	go e.git.Run(ctx)

	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	e.detectAll(ctx)
	for {
		select {
		case <-tick.C:
			e.detectAll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) TriggerDetection(ctx context.Context, clusterID string) {
	e.detect(ctx, clusterID)
}

// TriggerPollAndDetect is called by the webhook handler — re-polls git then re-detects
// for all clusters that have repos matching repoURL.
func (e *Engine) TriggerPollAndDetect(ctx context.Context, repoURL string) {
	e.git.TriggerPoll(ctx, repoURL)
	for _, c := range e.clusters.List() {
		e.detect(ctx, c.ID)
	}
}

func (e *Engine) GetLatest(clusterID string) []DriftEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.latest[clusterID]
}

func (e *Engine) detectAll(ctx context.Context) {
	for _, c := range e.clusters.List() {
		e.detect(ctx, c.ID)
	}
}

func (e *Engine) detect(ctx context.Context, clusterID string) {
	live, err := e.clusters.GetLiveState(ctx, clusterID)
	if err != nil {
		e.log.Error("get live state failed", zap.String("cluster", clusterID), zap.Error(err))
		return
	}
	desired := e.git.GetDesiredState(clusterID)

	events := diff(clusterID, live, desired)

	e.mu.Lock()
	e.latest[clusterID] = events
	e.mu.Unlock()

	for _, ev := range events {
		e.audit.Record(ctx, audit.Entry{
			ClusterID: clusterID,
			Action:    "drift_detected",
			Resource:  fmt.Sprintf("%s/%s/%s", ev.ResourceKind, ev.Namespace, ev.ResourceName),
		})
		if ev.Severity == SeverityCritical {
			e.alerter.Send(ctx, ev)
		}
	}

	if len(events) > 0 {
		e.log.Info("drift detected", zap.String("cluster", clusterID), zap.Int("events", len(events)))
	}
}

func diff(clusterID string, live, desired []unstructured.Unstructured) []DriftEvent {
	liveIdx := indexByKey(live)
	desiredIdx := indexByKey(desired)

	var events []DriftEvent

	for key, d := range desiredIdx {
		l, exists := liveIdx[key]
		if !exists {
			events = append(events, makeDriftEvent(clusterID, nil, &d, DriftMissing, SeverityCritical))
			continue
		}
		// compare relevant fields only — status and metadata noise excluded
		dSpec := extractSpec(d)
		lSpec := extractSpec(l)
		if !cmp.Equal(dSpec, lSpec) {
			events = append(events, makeDriftEvent(clusterID, &l, &d, DriftModified, SeverityWarning))
		}
	}

	for key, l := range liveIdx {
		if _, exists := desiredIdx[key]; !exists {
			events = append(events, makeDriftEvent(clusterID, &l, nil, DriftUnexpected, SeverityInfo))
		}
	}

	return events
}

func indexByKey(objs []unstructured.Unstructured) map[string]unstructured.Unstructured {
	idx := make(map[string]unstructured.Unstructured, len(objs))
	for _, o := range objs {
		key := fmt.Sprintf("%s/%s/%s", o.GetKind(), o.GetNamespace(), o.GetName())
		idx[key] = o
	}
	return idx
}

func extractSpec(obj unstructured.Unstructured) map[string]interface{} {
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	return spec
}

func makeDriftEvent(clusterID string, live, desired *unstructured.Unstructured, dt DriftType, sev Severity) DriftEvent {
	ev := DriftEvent{
		ClusterID: clusterID,
		DriftType: dt,
		Severity:  sev,
		DetectedAt: time.Now(),
	}

	ref := desired
	if ref == nil {
		ref = live
	}
	ev.ResourceKind = ref.GetKind()
	ev.ResourceName = ref.GetName()
	ev.Namespace = ref.GetNamespace()

	if desired != nil {
		b, _ := json.Marshal(desired.Object)
		ev.DesiredState = b
	}
	if live != nil {
		b, _ := json.Marshal(live.Object)
		ev.LiveState = b
	}
	if live != nil && desired != nil {
		d := cmp.Diff(extractSpec(*desired), extractSpec(*live))
		ev.Diff, _ = json.Marshal(d)
	}

	return ev
}
