package reconcile

import (
	"context"
	"fmt"

	"github.com/yourorg/gitops-drift-detector/internal/audit"
	"github.com/yourorg/gitops-drift-detector/internal/cluster"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"strings"
)

type Reconciler struct {
	clusters *cluster.Manager
	audit    *audit.Logger
	log      *zap.Logger
}

func NewReconciler(clusters *cluster.Manager, audit *audit.Logger, log *zap.Logger) *Reconciler {
	return &Reconciler{clusters: clusters, audit: audit, log: log}
}

type ReconcileRequest struct {
	ClusterID    string `json:"cluster_id"`
	ResourceKind string `json:"resource_kind"`
	ResourceName string `json:"resource_name"`
	Namespace    string `json:"namespace"`
	Manifest     string `json:"manifest"` // raw YAML from git
	Actor        string `json:"actor"`
}

func (r *Reconciler) Apply(ctx context.Context, req ReconcileRequest) error {
	var obj unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(req.Manifest), 4096)
	if err := decoder.Decode(&obj.Object); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}

	if err := r.clusters.Apply(ctx, req.ClusterID, &obj); err != nil {
		r.log.Error("reconcile apply failed",
			zap.String("cluster", req.ClusterID),
			zap.String("resource", req.ResourceName),
			zap.Error(err),
		)
		return fmt.Errorf("apply: %w", err)
	}

	r.audit.Record(ctx, audit.Entry{
		ClusterID: req.ClusterID,
		Action:    "reconciled",
		Actor:     req.Actor,
		Resource:  fmt.Sprintf("%s/%s/%s", req.ResourceKind, req.Namespace, req.ResourceName),
	})

	r.log.Info("reconciled",
		zap.String("cluster", req.ClusterID),
		zap.String("resource", req.ResourceName),
		zap.String("actor", req.Actor),
	)
	return nil
}
