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
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
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

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}
