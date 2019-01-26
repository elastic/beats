// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package util

import (
	"testing"

	v1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

func TestBuildMetadataEnricher(t *testing.T) {
	watcher := mockWatcher{}
	funcs := mockFuncs{}
	resource := &mockResource{
		uid:       "mockuid",
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
			"_module": common.MapStr{"label": "value", "pod": common.MapStr{"name": "enrich", "uid": "mockuid"}},
		},
	}, events)

	// Enrich a pod (metadata goes in root level)
	events = []common.MapStr{
		{"name": "unknown"},
		{"name": "enrich"},
	}
	enricher.isPod = true
	enricher.Enrich(events)

	assert.Equal(t, []common.MapStr{
		{"name": "unknown"},
		{
			"name":    "enrich",
			"uid":     "mockuid",
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
	meta := common.MapStr{
		"pod": common.MapStr{
			"name": obj.GetMetadata().GetName(),
			"uid":  obj.GetMetadata().GetUid(),
		},
	}
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
	name, namespace, uid string
	labels               map[string]string
}

func (r *mockResource) GetMetadata() *v1.ObjectMeta {
	return &v1.ObjectMeta{
		Uid:       &r.uid,
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
