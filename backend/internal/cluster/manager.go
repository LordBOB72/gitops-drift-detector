package cluster

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterClient struct {
	ID        string
	Name      string
	client    kubernetes.Interface
	dynamic   dynamic.Interface
}

type Manager struct {
	mu      sync.RWMutex
	clients map[string]*ClusterClient
	log     *zap.Logger
}

func NewManager(log *zap.Logger) *Manager {
	return &Manager{
		clients: make(map[string]*ClusterClient),
		log:     log,
	}
}

func (m *Manager) AddCluster(id, name, kubeconfig string) error {
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		// fall back to in-cluster config if kubeconfig is empty
		var ierr error
		cfg, ierr = rest.InClusterConfig()
		if ierr != nil {
			return fmt.Errorf("parse kubeconfig: %w", err)
		}
	}

	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("kubernetes client: %w", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("dynamic client: %w", err)
	}

	m.mu.Lock()
	m.clients[id] = &ClusterClient{ID: id, Name: name, client: kc, dynamic: dyn}
	m.mu.Unlock()
	m.log.Info("cluster registered", zap.String("name", name))
	return nil
}

func (m *Manager) RemoveCluster(id string) {
	m.mu.Lock()
	delete(m.clients, id)
	m.mu.Unlock()
}

func (m *Manager) List() []ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ClusterInfo, 0, len(m.clients))
	for _, c := range m.clients {
		out = append(out, ClusterInfo{ID: c.ID, Name: c.Name})
	}
	return out
}

func (m *Manager) GetLiveState(ctx context.Context, clusterID string) ([]unstructured.Unstructured, error) {
	m.mu.RLock()
	c, ok := m.clients[clusterID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	var resources []unstructured.Unstructured

	// fetch the resource types we care about
	gvrs := []schema.GroupVersionResource{
		{Group: "apps", Version: "v1", Resource: "deployments"},
		{Group: "", Version: "v1", Resource: "configmaps"},
		{Group: "", Version: "v1", Resource: "services"},
		{Group: "apps", Version: "v1", Resource: "statefulsets"},
		{Group: "apps", Version: "v1", Resource: "daemonsets"},
	}

	for _, gvr := range gvrs {
		list, err := c.dynamic.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
		if err != nil {
			m.log.Warn("list resources failed", zap.String("resource", gvr.Resource), zap.Error(err))
			continue
		}
		resources = append(resources, list.Items...)
	}

	return resources, nil
}

func (m *Manager) Apply(ctx context.Context, clusterID string, obj *unstructured.Unstructured) error {
	m.mu.RLock()
	c, ok := m.clients[clusterID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	gvr, err := resolveGVR(obj)
	if err != nil {
		return err
	}

	_, err = c.dynamic.Resource(gvr).Namespace(obj.GetNamespace()).Apply(
		ctx,
		obj.GetName(),
		obj,
		metav1.ApplyOptions{FieldManager: "gitops-drift-detector", Force: true},
	)
	return err
}

func resolveGVR(obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	gvk := obj.GroupVersionKind()
	// basic mapping — in production you'd use a REST mapper
	resourceMap := map[string]string{
		"Deployment":  "deployments",
		"ConfigMap":   "configmaps",
		"Service":     "services",
		"StatefulSet": "statefulsets",
		"DaemonSet":   "daemonsets",
	}
	res, ok := resourceMap[gvk.Kind]
	if !ok {
		return schema.GroupVersionResource{}, fmt.Errorf("unknown kind: %s", gvk.Kind)
	}
	return schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: res}, nil
}

// keep the compiler happy with the imports we declared at the top
var _ = corev1.Pod{}
var _ = appsv1.Deployment{}

type ClusterInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
