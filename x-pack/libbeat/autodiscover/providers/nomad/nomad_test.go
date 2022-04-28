// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/nomad/api"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/nomad"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
		// Scenarios being tested:
		// - logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// - metrics/module must be found in hints.metrics
		// - not.to.include must not be part of hints
		{
			event: bus.Event{
				"nomad": mapstr.M{
					"allocation": mapstr.M{
						"id": "cf7db85d-c93c-873a-cb37-6d2ea071b0eb",
					},
					"datacenter": []string{"europe-west4"},
				},
				"meta": mapstr.M{
					"nomad": mapstr.M{
						"task": getNestedAnnotations(mapstr.M{
							"allocation": mapstr.M{
								"id": "f67d087a-fb67-48a8-b526-ac1316f4bc9a",
							},
							"co.elastic.logs/multiline.pattern": "^test",
							"co.elastic.metrics/module":         "prometheus",
							"co.elastic.metrics/period":         "10s",
							"not.to.include":                    "true",
						}),
					},
				},
			},
			result: bus.Event{
				"nomad": mapstr.M{
					"task": getNestedAnnotations(mapstr.M{
						"allocation": mapstr.M{
							"id": "f67d087a-fb67-48a8-b526-ac1316f4bc9a",
						},
						"not.to.include": "true",
					}),
				},
				"hints": mapstr.M{
					"logs": mapstr.M{
						"multiline": mapstr.M{
							"pattern": "^test",
						},
					},
					"metrics": mapstr.M{
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
		logger: logp.NewLogger("nomad"),
	}
	for _, test := range tests {
		assert.Equal(t, test.result, p.generateHints(test.event))
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
				ID:            UUID.String(),
				Name:          "job.task",
				Namespace:     namespace,
				DesiredStatus: api.AllocDesiredStatusRun,
				ClientStatus:  api.AllocClientStatusRunning,
				NodeName:      host,
				NodeID:        "nomad1",
				Job: &nomad.Job{
					ID:          nomad.StringToPtr(UUID.String()),
					Region:      nomad.StringToPtr("global"),
					Name:        nomad.StringToPtr("my-job"),
					Type:        nomad.StringToPtr(nomad.JobTypeService),
					Status:      nomad.StringToPtr(nomad.JobStatusRunning),
					Datacenters: []string{"europe-west4"},
					Meta: map[string]string{
						"key1":    "job-value",
						"job-key": "job.value",
					},
					TaskGroups: []*nomad.TaskGroup{
						{
							Name: nomad.StringToPtr("web"),
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
				TaskStates: map[string]*api.TaskState{
					"task1": {
						State: nomad.TaskStateRunning,
					},
				},
			},
			Expected: bus.Event{
				"provider": UUID,
				"id":       fmt.Sprintf("%s-%s", UUID.String(), "task1"),
				"config":   []*common.Config{},
				"start":    true,
				"host":     host,
				"nomad": mapstr.M{
					"allocation": mapstr.M{
						"id":     UUID.String(),
						"name":   "job.task",
						"status": "running",
					},
					"datacenter": []string{"europe-west4"},
					"job": mapstr.M{
						"name": "my-job",
						"type": "service",
					},
					"namespace": "default",
					"region":    "global",
				},
				"meta": mapstr.M{
					"nomad": mapstr.M{
						"datacenter": []string{"europe-west4"},
						"job": mapstr.M{
							"name": "my-job",
							"type": "service",
						},
						"task": mapstr.M{
							"group-key": "group.value",
							"job-key":   "job.value",
							"key1":      "task-value",
							"name":      "task1",
							"service": mapstr.M{
								"name": []string{"web", "nginx"},
								"tags": []string{"tag-a", "tag-b", "tag-c", "tag-d"},
							},
							"task-key": "task.value",
						},
						"namespace": "default",
						"region":    "global",
						"allocation": mapstr.M{
							"id":     UUID.String(),
							"name":   "job.task",
							"status": nomad.AllocClientStatusRunning,
						},
					},
				},
			},
		},
		{
			Message: "Allocation without a host/node name",
			Status:  "start",
			Allocation: nomad.Resource{
				ID:            UUID.String(),
				Name:          "job.task",
				Namespace:     "default",
				DesiredStatus: api.AllocDesiredStatusRun,
				ClientStatus:  api.AllocClientStatusRunning,
				NodeName:      "",
				NodeID:        "5456bd7a",
				Job: &nomad.Job{
					ID:          nomad.StringToPtr(UUID.String()),
					Region:      nomad.StringToPtr("global"),
					Name:        nomad.StringToPtr("my-job"),
					Type:        nomad.StringToPtr(nomad.JobTypeService),
					Status:      nomad.StringToPtr(nomad.JobStatusRunning),
					Datacenters: []string{"europe-west4"},
					Meta: map[string]string{
						"key1":    "job-value",
						"job-key": "job.value",
					},
					TaskGroups: []*nomad.TaskGroup{
						{
							Name: nomad.StringToPtr("web"),
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

	config := &api.Config{
		Address:  "http://127.0.0.1",
		SecretID: "",
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// use the httpmock patched client
	config.HttpClient = http.DefaultClient

	httpmock.RegisterResponder(http.MethodGet, "http://127.0.0.1/v1/node/5456bd7a",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusNotFound, ""), nil
		},
	)

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		t.Run(test.Message, func(t *testing.T) {
			mapper, err := template.NewConfigMapper(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			metaGen, err := nomad.NewMetaGenerator(common.NewConfig(), client)
			if err != nil {
				t.Fatal(err)
			}

			p := &Provider{
				config:    defaultConfig(),
				bus:       bus.New(logp.NewLogger("bus"), "test"),
				metagen:   metaGen,
				templates: mapper,
				uuid:      UUID,
				logger:    logp.NewLogger("nomad"),
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

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	assert.Equal(t, httpmock.GetCallCountInfo()["GET http://127.0.0.1/v1/node/5456bd7a"], 1)
}

func getNestedAnnotations(in mapstr.M) mapstr.M {
	out := mapstr.M{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}
