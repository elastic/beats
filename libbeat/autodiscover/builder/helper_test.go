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
		annotations map[string]string
		result      common.MapStr
	}{
		// Empty annotations should return empty hints
		{
			annotations: map[string]string{},
			result:      common.MapStr{},
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
				"co.elastic.metrics.foobar1/period": "12s",
				"not.to.include":                    "true",
			},
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
	}

	for _, test := range tests {
		annMap := common.MapStr{}
		for k, v := range test.annotations {
			annMap.Put(k, v)
		}
		assert.Equal(t, test.result, GenerateHints(annMap, "foobar", "co.elastic", "/"))
	}
}

func TestGenerateHintsCustomSeparator(t *testing.T) {
	tests := []struct {
		annotations map[string]string
		result      common.MapStr
		separator   string
		container   string
	}{
		// Scenarios being tested: with . as separator
		// logs.include must be included under hints.logs
		// not_to_include must not be part of hints
		// not_to_include must not be part of hints.metrics
		// metrics.period must be set to 15s as container has higher priority than global hint
		// metrics.foobar1.period must be set as it can't be distinguished between container or hint
		{
			annotations: map[string]string{
				"co.elastic.logs.include":           "true",
				"not_to_include":                    "false",
				"co.elastic.metrics/not_to_include": "false",
				"co.elastic.metrics.period":         "10s",
				"co.elastic.metrics.foobar.period":  "15s",
				"co.elastic.metrics.foobar1.period": "12s",
			},
			container: "foobar",
			separator: "\\.",
			result: common.MapStr{
				"logs": common.MapStr{
					"include": "true",
				},
				"metrics": common.MapStr{
					"period": "15s",
					"foobar1": common.MapStr{
						"period": "12s",
					},
				},
			},
		},
		// Scenarios being tested: with . as separator and . in container
		// logs.include must be included under hints.logs
		// not_to_include must not be part of hints
		// not_to_include must not be part of hints.metrics
		// metrics.period must be set to 15s as container has higher priority than global hint
		// metrics.foo.bar1.period must be set as it can't be distinguished between container or hint
		{
			annotations: map[string]string{
				"co.elastic.logs.include":            "true",
				"not_to_include":                     "false",
				"co.elastic.metrics/not_to_include":  "false",
				"co.elastic.metrics.period":          "10s",
				"co.elastic.metrics.foo.bar.period":  "15s",
				"co.elastic.metrics.foo.bar1.period": "12s",
			},
			container: "foo.bar",
			separator: "\\.",
			result: common.MapStr{
				"logs": common.MapStr{
					"include": "true",
				},
				"metrics": common.MapStr{
					"period": "15s",
					"foo": common.MapStr{
						"bar1": common.MapStr{
							"period": "12s",
						},
					},
				},
			},
		},
		// Scenarios being tested: with - as separator and . in container
		// logs.include must be included under hints.logs
		// not_to_include must not be part of hints
		// not_to_include must not be part of hints.metrics
		// metrics.period must be set to 15s as container has higher priority than global hint
		// metrics.foo.bar1.period must not be set as it can be distinguished between container or hint
		{
			annotations: map[string]string{
				"co.elastic.logs-include":            "true",
				"not.to-include":                     "false",
				"co.elastic-metrics.not.to.include":  "false",
				"co.elastic.metrics-period":          "10s",
				"co.elastic.metrics.foo.bar-period":  "15s",
				"co.elastic.metrics.foo.bar1-period": "12s",
			},
			container: "foo.bar",
			separator: "-",
			result: common.MapStr{
				"logs": common.MapStr{
					"include": "true",
				},
				"metrics": common.MapStr{
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
		assert.Equal(t, test.result, GenerateHints(annMap, test.container, "co.elastic", test.separator))
	}
}
