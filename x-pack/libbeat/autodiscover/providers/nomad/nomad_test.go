// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/stretchr/testify/assert"
	"gopkg.in/jarcoal/httpmock.v1"

	"github.com/elastic/beats/libbeat/tests/resources"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/x-pack/libbeat/common/nomad"
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
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			event: bus.Event{
				"meta": common.MapStr{
					"task": getNestedAnnotations(common.MapStr{
						"alloc_id":                          "f67d087a-fb67-48a8-b526-ac1316f4bc9a",
						"co.elastic.logs/multiline.pattern": "^test",
						"co.elastic.metrics/module":         "prometheus",
						"co.elastic.metrics/period":         "10s",
						"not.to.include":                    "true",
					}),
				},
			},
			result: bus.Event{
				"meta": common.MapStr{
					"task": getNestedAnnotations(common.MapStr{
						"alloc_id":       "f67d087a-fb67-48a8-b526-ac1316f4bc9a",
						"not.to.include": "true",
					}),
				},
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
				ID:        UUID.String(),
				Name:      "job.task",
				Namespace: namespace,
				NodeName:  host,
				NodeID:    "nomad1",
				Job: &nomad.Job{
					ID:          helper.StringToPtr(UUID.String()),
					Region:      helper.StringToPtr("global"),
					Name:        helper.StringToPtr("my-job"),
					Type:        helper.StringToPtr(structs.JobTypeService),
					Status:      helper.StringToPtr(structs.AllocClientStatusRunning),
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
					"datacenters": []string{"europe-west4"},
					"job":         "my-job",
					"task": common.MapStr{
						"group-key": "group.value",
						"job-key":   "job.value",
						"key1":      "task-value",
						"name":      "task1",
						"service": common.MapStr{
							"canary_tags": []string{},
							"name":        []string{"web", "nginx"},
							"tags":        []string{"tag-a", "tag-b", "tag-c", "tag-d"},
						},
						"task-key": "task.value",
					},
					"name":      "job.task",
					"namespace": "default",
					"region":    "global",
					"type":      "service",
					"alloc_id":  UUID.String(),
					"status":    structs.AllocClientStatusRunning,
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
				NodeID:    "5456bd7a",
				Job: &nomad.Job{
					ID:          helper.StringToPtr(UUID.String()),
					Region:      helper.StringToPtr("global"),
					Name:        helper.StringToPtr("my-job"),
					Type:        helper.StringToPtr(structs.JobTypeService),
					Status:      helper.StringToPtr(structs.AllocClientStatusRunning),
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
			mapper, err := template.NewConfigMapper(nil)
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

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)


	assert.Equal(t, httpmock.GetCallCountInfo()["GET http://127.0.0.1/v1/node/5456bd7a"], 1)
}

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}
