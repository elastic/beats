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

package kubernetes

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/menderesk/beats/v7/libbeat/autodiscover/template"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/bus"
	"github.com/menderesk/beats/v7/libbeat/common/kubernetes"
	"github.com/menderesk/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

func TestGenerateHints_Node(t *testing.T) {
	tests := []struct {
		event  bus.Event
		result bus.Event
	}{
		// Empty events should return empty hints
		{
			event:  bus.Event{},
			result: bus.Event{},
		},
		// Only kubernetes payload must return only kubernetes as part of the hint
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"node": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"node": common.MapStr{
						"name": "foobar",
					},
				},
			},
		},
		// Scenarios being tested:
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"co.elastic.metrics/period": "10s",
						"not.to.include":            "true",
					}),
					"node": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"not.to.include":            "true",
						"co.elastic.metrics/period": "10s",
					}),
					"node": common.MapStr{
						"name": "foobar",
					},
				},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "10s",
					},
				},
			},
		},
	}

	cfg := defaultConfig()

	s := service{
		config: cfg,
		logger: logp.NewLogger("kubernetes.service"),
	}
	for _, test := range tests {
		assert.Equal(t, s.GenerateHints(test.event), test.result)
	}
}

func TestEmitEvent_Node(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	name := "metricbeat"
	nodeIP := "192.168.0.1"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	UUID, err := uuid.NewV4()

	typeMeta := metav1.TypeMeta{
		Kind:       "Node",
		APIVersion: "v1",
	}
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Message  string
		Flag     string
		Node     *kubernetes.Node
		Expected bus.Event
	}{
		{
			Message: "Test node start",
			Flag:    "start",
			Node: &kubernetes.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeExternalIP,
							Address: nodeIP,
						},
						{
							Type:    v1.NodeInternalIP,
							Address: "1.2.3.4",
						},
						{
							Type:    v1.NodeHostName,
							Address: "node1",
						},
					},
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			Expected: bus.Event{
				"start":    true,
				"host":     "192.168.0.1",
				"id":       uid,
				"provider": UUID,
				"kubernetes": common.MapStr{
					"node": common.MapStr{
						"name":     "metricbeat",
						"uid":      "005f3b90-4b9d-12f8-acf0-31020a840133",
						"hostname": "node1",
					},
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"node": common.MapStr{
							"name":     "metricbeat",
							"uid":      "005f3b90-4b9d-12f8-acf0-31020a840133",
							"hostname": "node1",
						},
					},
				},
				"config": []*common.Config{},
			},
		},
		{
			Message: "Test node start with just node name",
			Flag:    "start",
			Node: &kubernetes.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeHostName,
							Address: "node1",
						},
					},
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			Expected: bus.Event{
				"start":    true,
				"host":     "node1",
				"id":       uid,
				"provider": UUID,
				"kubernetes": common.MapStr{
					"node": common.MapStr{
						"name":     "metricbeat",
						"uid":      "005f3b90-4b9d-12f8-acf0-31020a840133",
						"hostname": "node1",
					},
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"node": common.MapStr{
							"name":     "metricbeat",
							"uid":      "005f3b90-4b9d-12f8-acf0-31020a840133",
							"hostname": "node1",
						},
					},
				},
				"config": []*common.Config{},
			},
		},
		{
			Message: "Test service without host",
			Flag:    "start",
			Node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status:   v1.NodeStatus{},
			},
			Expected: nil,
		},
		{
			Message: "Test stop node without host",
			Flag:    "stop",
			Node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			Expected: bus.Event{
				"stop":     true,
				"host":     "node1",
				"id":       uid,
				"provider": UUID,
				"kubernetes": common.MapStr{
					"node": common.MapStr{
						"name":     "metricbeat",
						"uid":      "005f3b90-4b9d-12f8-acf0-31020a840133",
						"hostname": "node1",
					},
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"node": common.MapStr{
							"name":     "metricbeat",
							"uid":      "005f3b90-4b9d-12f8-acf0-31020a840133",
							"hostname": "node1",
						},
					},
				},
				"config": []*common.Config{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Message, func(t *testing.T) {
			mapper, err := template.NewConfigMapper(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			metaGen := metadata.NewNodeMetadataGenerator(common.NewConfig(), nil, client)
			config := defaultConfig()
			p := &Provider{
				config:    config,
				bus:       bus.New(logp.NewLogger("bus"), "test"),
				templates: mapper,
				logger:    logp.NewLogger("kubernetes"),
			}

			no := &node{
				metagen: metaGen,
				config:  defaultConfig(),
				publish: p.publish,
				uuid:    UUID,
				logger:  logp.NewLogger("kubernetes.no"),
			}

			p.eventManager = NewMockNodeEventerManager(no)

			listener := p.bus.Subscribe()

			no.emit(test.Node, test.Flag)

			select {
			case event := <-listener.Events():
				assert.Equal(t, test.Expected, event, test.Message)
			case <-time.After(2 * time.Second):
				if test.Expected != nil {
					t.Fatal("Timeout while waiting for event")
				}
			}
		})
	}
}

func NewMockNodeEventerManager(no *node) EventManager {
	em := &eventerManager{}
	em.eventer = no
	return em
}

func TestNode_isUpdated(t *testing.T) {
	tests := []struct {
		old     *kubernetes.Node
		new     *kubernetes.Node
		updated bool
		test    string
	}{
		{
			test:    "one of the objects is nil then its updated",
			old:     nil,
			new:     &kubernetes.Node{},
			updated: true,
		},
		{
			test:    "both empty nodes should return not updated",
			old:     &kubernetes.Node{},
			new:     &kubernetes.Node{},
			updated: false,
		},
		{
			test: "resource version is the same should return not updated",
			old: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "1",
				},
			},
			new: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "1",
				},
			},
		},
		{
			test: "if meta changes then it should return updated",
			old: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "1",
					Annotations:     map[string]string{},
				},
			},
			new: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "2",
					Annotations: map[string]string{
						"a": "b",
					},
				},
			},
			updated: true,
		},
		{
			test: "if spec changes then it should return updated",
			old: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"a": "b",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID:    "1",
					Unschedulable: false,
				},
			},
			new: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "2",
					Annotations: map[string]string{
						"a": "b",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID:    "1",
					Unschedulable: true,
				},
			},
			updated: true,
		},
		{
			test: "if overall status doesn't change then its not an update",
			old: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"a": "b",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID:    "1",
					Unschedulable: true,
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			new: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "2",
					Annotations: map[string]string{
						"a": "b",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID:    "1",
					Unschedulable: true,
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			updated: false,
		},
		{
			test: "if node status changes then its an update",
			old: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "1",
					Annotations: map[string]string{
						"a": "b",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID:    "1",
					Unschedulable: true,
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			new: &kubernetes.Node{
				ObjectMeta: kubernetes.ObjectMeta{
					ResourceVersion: "2",
					Annotations: map[string]string{
						"a": "b",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID:    "1",
					Unschedulable: true,
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			updated: true,
		},
	}

	for _, test := range tests {
		t.Run(test.test, func(t *testing.T) {
			assert.Equal(t, test.updated, isUpdated(test.old, test.new))
		})
	}
}
