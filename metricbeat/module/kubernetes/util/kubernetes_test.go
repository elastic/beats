package util

import (
	"testing"

	"github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

func TestBuildMetadataEnricher(t *testing.T) {
	watcher := mockWatcher{}
	funcs := mockFuncs{}
	resource := &mockResource{
		name:      "enrich",
		namespace: "default",
		labels: map[string]string{
			"label": "value",
		},
	}

	enricher := buildMetadataEnricher(&watcher, funcs.update, funcs.delete, funcs.index)
	assert.NotNil(t, watcher.handler)

	enricher.Start()
	assert.True(t, watcher.started)

	// Emit an event
	watcher.handler.OnAdd(resource)
	assert.Equal(t, resource, funcs.updated)

	// Test enricher
	events := []common.MapStr{
		{"name": "unknown"},
		{"name": "enrich"},
	}
	enricher.Enrich(events)

	assert.Equal(t, []common.MapStr{
		{"name": "unknown"},
		{
			"name":    "enrich",
			"_module": common.MapStr{"label": "value"},
		},
	}, events)

	// Emit delete event
	watcher.handler.OnDelete(resource)
	assert.Equal(t, resource, funcs.deleted)

	events = []common.MapStr{
		{"name": "unknown"},
		{"name": "enrich"},
	}
	enricher.Enrich(events)

	assert.Equal(t, []common.MapStr{
		{"name": "unknown"},
		{"name": "enrich"},
	}, events)
}

type mockFuncs struct {
	updated kubernetes.Resource
	deleted kubernetes.Resource
	indexed common.MapStr
}

func (f *mockFuncs) update(m map[string]common.MapStr, obj kubernetes.Resource) {
	f.updated = obj
	meta := common.MapStr{}
	for k, v := range obj.GetMetadata().Labels {
		meta[k] = v
	}
	m[obj.GetMetadata().GetName()] = meta
}

func (f *mockFuncs) delete(m map[string]common.MapStr, obj kubernetes.Resource) {
	f.deleted = obj
	delete(m, obj.GetMetadata().GetName())
}

func (f *mockFuncs) index(m common.MapStr) string {
	f.indexed = m
	return m["name"].(string)
}

type mockResource struct {
	name, namespace string
	labels          map[string]string
}

func (r *mockResource) GetMetadata() *v1.ObjectMeta {
	return &v1.ObjectMeta{
		Name:      &r.name,
		Namespace: &r.namespace,
		Labels:    r.labels,
	}
}

type mockWatcher struct {
	handler kubernetes.ResourceEventHandler
	started bool
}

func (m *mockWatcher) Start() error {
	m.started = true
	return nil
}

func (m *mockWatcher) Stop() {

}
func (m *mockWatcher) AddEventHandler(r kubernetes.ResourceEventHandler) {
	m.handler = r
}
