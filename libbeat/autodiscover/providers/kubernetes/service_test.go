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

	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestGenerateHints_Service(t *testing.T) {
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
					"service": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"service": common.MapStr{
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
					"service": common.MapStr{
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
					"service": common.MapStr{
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
		// Scenarios tested:
		// Have one set of annotations come from service and the other from namespace defaults
		// The resultant should have both
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"not.to.include":            "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/period": "10s",
					}),
					"service": common.MapStr{
						"name": "foobar",
					},
					"namespace": "ns",
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"not.to.include":            "true",
					}),
					"service": common.MapStr{
						"name": "foobar",
					},
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/period": "10s",
					}),
					"namespace": "ns",
				},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "10s",
					},
				},
			},
		},
		// Scenarios tested:
		// Have the same set of annotations come from both namespace and service.
		// The resultant should have the ones from service alone
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"co.elastic.metrics/period": "10s",
						"not.to.include":            "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "dropwizard",
						"co.elastic.metrics/period": "60s",
					}),
					"namespace": "ns",
					"service": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"co.elastic.metrics/period": "10s",
						"not.to.include":            "true",
					}),
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "dropwizard",
						"co.elastic.metrics/period": "60s",
					}),
					"namespace": "ns",
					"service": common.MapStr{
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
		// Scenarios tested:
		// Have no annotations on the service and only have namespace level defaults
		// The resultant should have honored the namespace defaults
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"co.elastic.metrics/period": "10s",
					}),
					"service": common.MapStr{
						"name": "foobar",
					},
					"namespace": "ns",
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"namespace_annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.metrics/module": "prometheus",
						"co.elastic.metrics/period": "10s",
					}),
					"service": common.MapStr{
						"name": "foobar",
					},
					"namespace": "ns",
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

func TestEmitEvent_Service(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	name := "metricbeat"
	namespace := "default"
	clusterIP := "192.168.0.1"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	UUID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	typeMeta := metav1.TypeMeta{
		Kind:       "Service",
		APIVersion: "v1",
	}

	tests := []struct {
		Message  string
		Flag     string
		Service  *kubernetes.Service
		Expected bus.Event
	}{
		{
			Message: "Test service start",
			Flag:    "start",
			Service: &kubernetes.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name: "http",
							Port: 8080,
						},
					},
					ClusterIP: clusterIP,
				},
			},
			Expected: bus.Event{
				"start":    true,
				"host":     "192.168.0.1",
				"id":       uid,
				"provider": UUID,
				"kubernetes": common.MapStr{
					"service": common.MapStr{
						"name": "metricbeat",
						"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
					},
					"namespace":   "default",
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"namespace": "default",
						"service": common.MapStr{
							"name": "metricbeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
					},
				},
				"config": []*config.C{},
			},
		},
		{
			Message: "Test service without host",
			Flag:    "start",
			Service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name: "http",
							Port: 8080,
						},
					},
				},
			},
			Expected: nil,
		},
		{
			Message: "Test service without port",
			Flag:    "start",
			Service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Spec: v1.ServiceSpec{
					ClusterIP: clusterIP,
				},
			},
			Expected: nil,
		},
		{
			Message: "Test stop service without host",
			Flag:    "stop",
			Service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					UID:         types.UID(uid),
					Namespace:   namespace,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				TypeMeta: typeMeta,
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{
							Name: "http",
							Port: 8080,
						},
					},
				},
			},
			Expected: bus.Event{
				"stop":     true,
				"host":     "",
				"id":       uid,
				"provider": UUID,
				"kubernetes": common.MapStr{
					"service": common.MapStr{
						"name": "metricbeat",
						"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
					},
					"namespace":   "default",
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"namespace": "default",
						"service": common.MapStr{
							"name": "metricbeat",
							"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
						},
					},
				},
				"config": []*config.C{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Message, func(t *testing.T) {
			mapper, err := template.NewConfigMapper(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			metaGen := metadata.NewServiceMetadataGenerator(config.NewConfig(), nil, nil, client)

			p := &Provider{
				config:    defaultConfig(),
				bus:       bus.New(logp.NewLogger("bus"), "test"),
				templates: mapper,
				logger:    logp.NewLogger("kubernetes"),
			}

			service := &service{
				metagen: metaGen,
				config:  defaultConfig(),
				publish: p.publish,
				uuid:    UUID,
				logger:  logp.NewLogger("kubernetes.service"),
			}

			p.eventManager = NewMockServiceEventerManager(service)
			listener := p.bus.Subscribe()

			service.emit(test.Service, test.Flag)

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

func NewMockServiceEventerManager(svc *service) EventManager {
	em := &eventerManager{}
	em.eventer = svc
	return em
}
