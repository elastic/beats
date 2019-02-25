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

package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		annotations    map[string]string
		defaultDisable bool
		result         common.MapStr
	}{
		// Empty annotations should return empty hints
		{
			annotations:    map[string]string{},
			defaultDisable: false,
			result:         common.MapStr{},
		},

		// Scenarios being tested:
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			annotations: map[string]string{
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			defaultDisable: false,
			result: common.MapStr{
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
		},
		// Scenarios being tested:
		// logs.disable must be generated when defaultDisable is set and annotations does not
		// have co.elastic.logs/disable set to false.
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			annotations: map[string]string{
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			defaultDisable: true,
			result: common.MapStr{
				"logs": common.MapStr{
					"multiline": common.MapStr{
						"pattern": "^test",
					},
					"disable": "true",
				},
				"metrics": common.MapStr{
					"module": "prometheus",
					"period": "15s",
				},
			},
		},
		// Scenarios being tested:
		// logs.disable must not be generated when defaultDisable is set, but annotations
		// have co.elastic.logs/disable set to false.
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			annotations: map[string]string{
				"co.elastic.logs/disable":           "false",
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			defaultDisable: true,
			result: common.MapStr{
				"logs": common.MapStr{
					"multiline": common.MapStr{
						"pattern": "^test",
					},
					"disable": "false",
				},
				"metrics": common.MapStr{
					"module": "prometheus",
					"period": "15s",
				},
			},
		},
	}

	for _, test := range tests {
		annMap := common.MapStr{}
		for k, v := range test.annotations {
			annMap.Put(k, v)
		}
		assert.Equal(t, GenerateHints(annMap, "foobar", "co.elastic", test.defaultDisable), test.result)
	}
}
