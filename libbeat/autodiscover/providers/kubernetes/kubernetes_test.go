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

	v1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

func TestGenerateHints(t *testing.T) {
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
					"pod": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "foobar",
					},
				},
			},
		},
		// Kubernetes payload with container info must be bubbled to top level
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
				"container": common.MapStr{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "rkt",
				},
			},
		},
		// Scenarios being tested:
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern": "^test",
						"co.elastic.metrics/module":         "prometheus",
						"co.elastic.metrics/period":         "10s",
						"co.elastic.metrics.foobar/period":  "15s",
						"not.to.include":                    "true",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern": "^test",
						"co.elastic.metrics/module":         "prometheus",
						"not.to.include":                    "true",
						"co.elastic.metrics/period":         "10s",
						"co.elastic.metrics.foobar/period":  "15s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
						},
					},
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": common.MapStr{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "docker",
				},
			},
		},
	}

	cfg := defaultConfig()

	p := Provider{
		config: cfg,
	}
	for _, test := range tests {
		assert.Equal(t, p.generateHints(test.event), test.result)
	}
}

func TestEmitEvent(t *testing.T) {
	name := "filebeat"
	namespace := "default"
	podIP := "127.0.0.1"
	containerID := "docker://foobar"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	containerImage := "elastic/filebeat:6.3.0"
	node := "node"
	UUID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Message  string
		Flag     string
		Pod      *kubernetes.Pod
		Expected bus.Event
	}{
		{
			Message: "Test common pod start",
			Flag:    "start",
			Pod: &v1.Pod{
				Metadata: &metav1.ObjectMeta{
					Name:        &name,
					Uid:         &uid,
					Namespace:   &namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Status: &v1.PodStatus{
					PodIP: &podIP,
					ContainerStatuses: []*kubernetes.PodContainerStatus{
						{
							Name:        &name,
							ContainerID: &containerID,
						},
					},
				},
				Spec: &v1.PodSpec{
					NodeName: &node,
					Containers: []*kubernetes.Container{
						{
							Image: &containerImage,
							Name:  &name,
						},
					},
				},
			},
			Expected: bus.Event{
				"start":    true,
				"host":     "127.0.0.1",
				"id":       "foobar",
				"provider": UUID,
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"id":      "foobar",
						"name":    "filebeat",
						"image":   "elastic/filebeat:6.3.0",
						"runtime": "docker",
					},
					"pod": common.MapStr{
						"name": "filebeat",
						"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
					},
					"node": common.MapStr{
						"name": "node",
					},
					"namespace":   "default",
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"namespace": "default",
						"container": common.MapStr{
							"name": "filebeat",
						}, "pod": common.MapStr{
							"name": "filebeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						}, "node": common.MapStr{
							"name": "node",
						},
					},
				},
				"config": []*common.Config{},
			},
		},
		{
			Message: "Test pod without host",
			Flag:    "start",
			Pod: &v1.Pod{
				Metadata: &metav1.ObjectMeta{
					Name:        &name,
					Uid:         &uid,
					Namespace:   &namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Status: &v1.PodStatus{
					ContainerStatuses: []*kubernetes.PodContainerStatus{
						{
							Name:        &name,
							ContainerID: &containerID,
						},
					},
				},
				Spec: &v1.PodSpec{
					NodeName: &node,
					Containers: []*kubernetes.Container{
						{
							Image: &containerImage,
							Name:  &name,
						},
					},
				},
			},
			Expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Message, func(t *testing.T) {
			mapper, err := template.NewConfigMapper(nil)
			if err != nil {
				t.Fatal(err)
			}

			metaGen, err := kubernetes.NewMetaGenerator(common.NewConfig())
			if err != nil {
				t.Fatal(err)
			}

			p := &Provider{
				config:    defaultConfig(),
				bus:       bus.New("test"),
				metagen:   metaGen,
				templates: mapper,
				uuid:      UUID,
			}

			listener := p.bus.Subscribe()

			p.emit(test.Pod, test.Flag)

			select {
			case event := <-listener.Events():
				assert.Equal(t, test.Expected, event)
			case <-time.After(2 * time.Second):
				if test.Expected != nil {
					t.Fatal("Timeout while waiting for event")
				}
			}
		})
	}
}

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}
