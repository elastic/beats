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
	"fmt"
	"testing"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/kubernetes"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

var (
	logger = logp.NewLogger("kubernetes")
)

func TestBuildMetadataEnricher(t *testing.T) {
	watcher := mockWatcher{}
	nodeWatcher := mockWatcher{}
	namespaceWatcher := mockWatcher{}
	funcs := mockFuncs{}
	resource := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID("mockuid"),
			Name: "enrich",
			Labels: map[string]string{
				"label": "value",
			},
			Namespace: "default",
		},
	}

	enricher := buildMetadataEnricher(&watcher, &nodeWatcher, &namespaceWatcher, funcs.update, funcs.delete, funcs.index)
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
			"meta":    common.MapStr{"orchestrator": common.MapStr{"cluster": common.MapStr{"name": "gke-4242"}}},
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
			"meta":    common.MapStr{"orchestrator": common.MapStr{"cluster": common.MapStr{"name": "gke-4242"}}},
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
	accessor, _ := meta.Accessor(obj)
	f.updated = obj
	meta := common.MapStr{
		"kubernetes": common.MapStr{
			"pod": common.MapStr{
				"name": accessor.GetName(),
				"uid":  string(accessor.GetUID()),
			},
		},
	}
	for k, v := range accessor.GetLabels() {
		ShouldPut(meta, fmt.Sprintf("kubernetes.%v", k), v, logger)
	}
	ShouldPut(meta, "orchestrator.cluster.name", "gke-4242", logger)
	m[accessor.GetName()] = meta
}

func (f *mockFuncs) delete(m map[string]common.MapStr, obj kubernetes.Resource) {
	accessor, _ := meta.Accessor(obj)
	f.deleted = obj
	delete(m, accessor.GetName())
}

func (f *mockFuncs) index(m common.MapStr) string {
	f.indexed = m
	return m["name"].(string)
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

func (m *mockWatcher) Store() cache.Store {
	return nil
}

func (m *mockWatcher) Client() k8s.Interface {
	return nil
}
