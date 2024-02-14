package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"strings"
)

type Repo struct {
	ID        string
	ClusterID string
	URL       string
	Branch    string
	Token     string
	Path      string
}

type Poller struct {
	interval time.Duration
	mu       sync.RWMutex
	repos    map[string]*Repo
	cache    map[string][]unstructured.Unstructured // repoID -> parsed manifests
	log      *zap.Logger
}

func NewPoller(interval time.Duration, log *zap.Logger) *Poller {
	return &Poller{
		interval: interval,
		repos:    make(map[string]*Repo),
		cache:    make(map[string][]unstructured.Unstructured),
		log:      log,
	}
}

func (p *Poller) AddRepo(r *Repo) {
	p.mu.Lock()
	p.repos[r.ID] = r
	p.mu.Unlock()
}

func (p *Poller) RemoveRepo(id string) {
	p.mu.Lock()
	delete(p.repos, id)
	delete(p.cache, id)
	p.mu.Unlock()
}

func (p *Poller) GetDesiredState(clusterID string) []unstructured.Unstructured {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var all []unstructured.Unstructured
	for _, r := range p.repos {
		if r.ClusterID != clusterID {
			continue
		}
		all = append(all, p.cache[r.ID]...)
	}
	return all
}

// Run polls all repos on the configured interval. Call in a goroutine.
func (p *Poller) Run(ctx context.Context) {
	tick := time.NewTicker(p.interval)
	defer tick.Stop()
	// poll immediately on start
	p.pollAll(ctx)
	for {
		select {
		case <-tick.C:
			p.pollAll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// TriggerPoll is called by the webhook handler for push-based updates
func (p *Poller) TriggerPoll(ctx context.Context, repoURL string) {
	p.mu.RLock()
	var ids []string
	for id, r := range p.repos {
		if r.URL == repoURL {
			ids = append(ids, id)
		}
	}
	p.mu.RUnlock()

	for _, id := range ids {
		p.pollRepo(ctx, id)
	}
}

func (p *Poller) pollAll(ctx context.Context) {
	p.mu.RLock()
	ids := make([]string, 0, len(p.repos))
	for id := range p.repos {
		ids = append(ids, id)
	}
	p.mu.RUnlock()

	for _, id := range ids {
		p.pollRepo(ctx, id)
	}
}

func (p *Poller) pollRepo(ctx context.Context, id string) {
	p.mu.RLock()
	r, ok := p.repos[id]
	p.mu.RUnlock()
	if !ok {
		return
	}

	dir, err := os.MkdirTemp("", "gitops-*")
	if err != nil {
		p.log.Error("mktemp failed", zap.Error(err))
		return
	}
	defer os.RemoveAll(dir)

	cloneOpts := &gogit.CloneOptions{
		URL:           r.URL,
		ReferenceName: plumbing.NewBranchReferenceName(r.Branch),
		SingleBranch:  true,
		Depth:         1,
	}
	if r.Token != "" {
		cloneOpts.Auth = &http.BasicAuth{Username: "x-token", Password: r.Token}
	}

	if _, err := gogit.PlainCloneContext(ctx, dir, false, cloneOpts); err != nil {
		p.log.Error("git clone failed", zap.String("repo", r.URL), zap.Error(err))
		return
	}

	manifests, err := parseManifests(filepath.Join(dir, r.Path))
	if err != nil {
		p.log.Error("manifest parse failed", zap.String("repo", r.URL), zap.Error(err))
		return
	}

	p.mu.Lock()
	p.cache[id] = manifests
	p.mu.Unlock()
	p.log.Info("repo polled", zap.String("repo", r.URL), zap.Int("manifests", len(manifests)))
}

func parseManifests(dir string) ([]unstructured.Unstructured, error) {
	var out []unstructured.Unstructured

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(data)), 4096)
		for {
			var obj unstructured.Unstructured
			if err := decoder.Decode(&obj.Object); err != nil {
				break
			}
			if obj.Object == nil {
				continue
			}
			out = append(out, obj)
		}
		return nil
	})

	return out, err
}
