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

package nomad

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/nomad"
	"github.com/gofrs/uuid"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/stretchr/testify/assert"
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
				"meta": common.MapStr{
					"task1": common.MapStr{
						"group-key": "group",
						"job-key":   "job",
						"task-key":  "task",
					},
				},
			},
			result: bus.Event{
				"meta": common.MapStr{
					"task1": common.MapStr{
						"group-key": "group",
						"job-key":   "job",
						"task-key":  "task",
					},
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
				"meta": getNestedAnnotations(common.MapStr{
					"co.elastic.logs/multiline.pattern": "^test",
					"co.elastic.metrics/module":         "prometheus",
					"co.elastic.metrics/period":         "10s",
					"not.to.include":                    "true",
				}),
			},
			result: bus.Event{
				"meta": getNestedAnnotations(common.MapStr{
					"co.elastic.logs/multiline.pattern": "^test",
					"co.elastic.metrics/module":         "prometheus",
					"co.elastic.metrics/period":         "10s",
					"not.to.include":                    "true",
				}),
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
						},
					},
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "10s",
					},
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
	host := "nomad1"
	namespace := "default"

	UUID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Message    string
		Status     string
		Allocation nomad.Resource
		Expected   bus.Event
	}{
		{
			Message: "Test common allocation start",
			Status:  "start",
			Allocation: nomad.Resource{
				ID:        UUID.String(),
				Name:      "job.task",
				Namespace: namespace,
				NodeName:  host,
				Job: &nomad.Job{
					ID:          helper.StringToPtr(UUID.String()),
					Region:      helper.StringToPtr("global"),
					Name:        helper.StringToPtr("my-job"),
					Type:        helper.StringToPtr(structs.JobTypeService),
					Datacenters: []string{"europe-west4"},
					Meta: map[string]string{
						"key1":    "job-value",
						"job-key": "job.value",
					},
					TaskGroups: []*nomad.TaskGroup{
						{
							Name: helper.StringToPtr("web"),
							Meta: map[string]string{
								"key1":      "group-value",
								"group-key": "group.value",
							},
							Tasks: []*api.Task{
								{
									Name: "task1",
									Meta: map[string]string{
										"key1":     "task-value",
										"task-key": "task.value",
									},
									Services: []*api.Service{
										{
											Id:   "service-a",
											Name: "web",
											Tags: []string{"tag-a", "tag-b"},
										},
										{
											Id:   "service-b",
											Name: "nginx",
											Tags: []string{"tag-c", "tag-d"},
										},
									},
								},
							},
						},
					},
				},
			},
			Expected: bus.Event{
				"provider": UUID,
				"id":       UUID.String(),
				"config":   []*common.Config{},
				"start":    true,
				"host":     host,
				"meta": common.MapStr{
					"nomad": common.MapStr{
						"datacenters": []string{"europe-west4"},
						"job":         "my-job",
						"meta": []common.MapStr{
							common.MapStr{
								"task1": common.MapStr{
									"canary_tags": []string{"web", "nginx"},
									"group-key":   "group.value",
									"job-key":     "job.value",
									"key1":        "task-value",
									"name":        []string{"web", "nginx"},
									"tags":        []string{"web", "nginx", "tag-c", "tag-d"},
									"task-key":    "task.value"},
							},
						},
						"name":      "job.task",
						"namespace": "default",
						"region":    "global",
						"type":      "service",
						"uuid":      UUID.String(),
					},
				},
			},
		},
		{
			Message: "Allocation without a host/node name",
			Status:  "start",
			Allocation: nomad.Resource{
				ID:        UUID.String(),
				Name:      "job.task",
				Namespace: "default",
				NodeName:  "",
				Job: &nomad.Job{
					ID:          helper.StringToPtr(UUID.String()),
					Region:      helper.StringToPtr("global"),
					Name:        helper.StringToPtr("my-job"),
					Type:        helper.StringToPtr(structs.JobTypeService),
					Datacenters: []string{"europe-west4"},
					Meta: map[string]string{
						"key1":    "job-value",
						"job-key": "job.value",
					},
					TaskGroups: []*nomad.TaskGroup{
						{
							Name: helper.StringToPtr("web"),
							Meta: map[string]string{
								"key1":      "group-value",
								"group-key": "group.value",
							},
							Tasks: []*api.Task{
								{
									Name: "task1",
									Meta: map[string]string{
										"key1":     "task-value",
										"task-key": "task.value",
									},
									Services: []*api.Service{
										{
											Id:   "service-a",
											Name: "web",
											Tags: []string{"tag-a", "tag-b"},
										},
										{
											Id:   "service-b",
											Name: "nginx",
											Tags: []string{"tag-c", "tag-d"},
										},
									},
								},
							},
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

			client, err := makeClient()
			if err != nil {
				t.Fatal(err)
			}

			metaGen, err := nomad.NewMetaGenerator(common.NewConfig(), client)
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

			p.emit(&test.Allocation, test.Status)

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

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}

func makeClient() (*nomad.Client, error) {
	config := api.DefaultConfig()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	config.Address = ts.URL

	return api.NewClient(config)
}
