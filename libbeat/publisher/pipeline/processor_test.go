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

package pipeline

import (
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestProcessors(t *testing.T) {
	defaultInfo := beat.Info{}

	type local struct {
		config               beat.ClientConfig
		events               []common.MapStr
		expected             []common.MapStr
		includeAgentMetadata bool
	}

	tests := []struct {
		name   string
		global pipelineProcessors
		local  []local
		info   *beat.Info
	}{
		{
			name: "user global fields and tags",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			local: []local{
				{
					config: beat.ClientConfig{},
					events: []common.MapStr{{"value": "abc", "user": nil}},
					expected: []common.MapStr{
						{"value": "abc", "global": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			name: "no normalization",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			local: []local{
				{
					config: beat.ClientConfig{SkipNormalization: true},
					events: []common.MapStr{{"value": "abc", "user": nil}},
					expected: []common.MapStr{
						{"value": "abc", "user": nil, "global": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			name: "add agent metadata",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 1, "agent": common.MapStr{"foo": "bar"}},
				tags:   []string{"tag"},
			},
			info: &beat.Info{
				Beat:        "test",
				EphemeralID: uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000")),
				Hostname:    "test.host.name",
				ID:          uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440001")),
				Name:        "test.host.name",
				Version:     "0.1",
			},
			local: []local{
				{
					config: beat.ClientConfig{},
					events: []common.MapStr{{"value": "abc", "user": nil}},
					expected: []common.MapStr{
						{
							"agent": common.MapStr{
								"ephemeral_id": "123e4567-e89b-12d3-a456-426655440000",
								"hostname":     "test.host.name",
								"id":           "123e4567-e89b-12d3-a456-426655440001",
								"type":         "test",
								"version":      "0.1",
								"foo":          "bar",
							},
							"value": "abc", "global": 1, "tags": []string{"tag"},
						},
					},
					includeAgentMetadata: true,
				},
			},
		},
		{
			name: "add agent metadata with custom host.name",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			info: &beat.Info{
				Beat:        "test",
				EphemeralID: uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000")),
				Hostname:    "test.host.name",
				ID:          uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440001")),
				Name:        "other.test.host.name",
				Version:     "0.1",
			},
			local: []local{
				{
					config: beat.ClientConfig{},
					events: []common.MapStr{{"value": "abc", "user": nil}},
					expected: []common.MapStr{
						{
							"agent": common.MapStr{
								"ephemeral_id": "123e4567-e89b-12d3-a456-426655440000",
								"hostname":     "test.host.name",
								"id":           "123e4567-e89b-12d3-a456-426655440001",
								"name":         "other.test.host.name",
								"type":         "test",
								"version":      "0.1",
							},
							"value": "abc", "global": 1, "tags": []string{"tag"},
						},
					},
					includeAgentMetadata: true,
				},
			},
		},
		{
			name: "beat local fields",
			local: []local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "local": 1}},
				},
			},
		},
		{
			name: "beat local and user global fields",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			local: []local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "local": 1, "global": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			name: "user global fields overwrite beat local fields",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 1, "shared": "global"},
				tags:   []string{"tag"},
			},
			local: []local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1, "shared": "local"},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "local": 1, "global": 1, "shared": "global", "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			name: "beat local fields isolated",
			local: []local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "local": 1}},
				},
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 2},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "local": 2}},
				},
			},
		},

		{
			name: "beat local fields + user global fields isolated",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 0},
			},
			local: []local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "global": 0, "local": 1}},
				},
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 2},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "global": 0, "local": 2}},
				},
			},
		},
		{
			name: "user local fields and tags",
			local: []local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 1},
							Tags:   []string{"tag"},
						},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "fields": common.MapStr{"local": 1}, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			name: "user local fields (under root) and tags",
			local: []local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 1},
							FieldsUnderRoot: true,
							Tags:            []string{"tag"},
						},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "local": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			name: "user local fields overwrite user global fields",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 0, "shared": "global"},
				tags:   []string{"global"},
			},
			local: []local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 1, "shared": "local"},
							FieldsUnderRoot: true,
							Tags:            []string{"local"},
						},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{
							"value":  "abc",
							"global": 0, "local": 1, "shared": "local",
							"tags": []string{"global", "local"},
						},
					},
				},
			},
		},
		{
			name: "user local fields isolated",
			local: []local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 1},
						},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "fields": common.MapStr{"local": 1}}},
				},
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 2},
						},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "fields": common.MapStr{"local": 2}}},
				},
			},
		},
		{
			name: "user local + global fields isolated",
			global: pipelineProcessors{
				fields: common.MapStr{"fields": common.MapStr{"global": 0}},
			},
			local: []local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 1},
						},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "fields": common.MapStr{"global": 0, "local": 1}}},
				},
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 2},
						},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "fields": common.MapStr{"global": 0, "local": 2}}},
				},
			},
		},
		{
			name: "user local + global fields isolated (fields with root)",
			global: pipelineProcessors{
				fields: common.MapStr{"global": 0},
			},
			local: []local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 1},
							FieldsUnderRoot: true,
						},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "global": 0, "local": 1}},
				},
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 2},
							FieldsUnderRoot: true,
						},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "global": 0, "local": 2}},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			monitors := Monitors{
				Logger: logp.NewLogger("test processors"),
			}

			// create processor pipelines
			programs := make([]beat.Processor, len(test.local))
			info := defaultInfo
			if test.info != nil {
				info = *test.info
			}
			for i, local := range test.local {
				local.config.SkipAgentMetadata = !local.includeAgentMetadata
				programs[i] = newProcessorPipeline(info, monitors, test.global, local.config)
			}

			// run processor pipelines in parallel
			var (
				wg      sync.WaitGroup
				mux     sync.Mutex
				results = make([][]common.MapStr, len(programs))
			)
			for id, local := range test.local {
				wg.Add(1)
				id, program, local := id, programs[id], local
				go func() {
					defer wg.Done()

					actual := make([]common.MapStr, len(local.events))
					for i, event := range local.events {
						out, _ := program.Run(&beat.Event{
							Timestamp: time.Now(),
							Fields:    event,
						})
						actual[i] = out.Fields
					}

					mux.Lock()
					defer mux.Unlock()
					results[id] = actual
				}()
			}
			wg.Wait()

			// validate
			for i, local := range test.local {
				assert.Equal(t, local.expected, results[i])
			}
		})
	}
}
