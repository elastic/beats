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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestGetProcessors(t *testing.T) {
	hints := mapstr.M{
		"co": mapstr.M{
			"elastic": mapstr.M{
				"logs": mapstr.M{
					"processors": mapstr.M{
						"add_fields": `{"fields": {"foo": "bar"}}`,
					},
				},
			},
		},
	}
	procs := GetProcessors(hints, "co.elastic.logs")
	assert.Equal(t, []mapstr.M{
		mapstr.M{
			"add_fields": mapstr.M{
				"fields": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	}, procs)
}

func TestGenerateHints(t *testing.T) {

	var allSupportedHints = []string{"enabled", "module", "integration", "datas_treams", "host", "period", "timeout", "metrics_path", "username", "password", "stream", "processors", "multiline", "json", "disable"}

	tests := []struct {
		name                   string
		annotations            map[string]string
		result                 mapstr.M
		expectedIncorrectHints int // We set the number of hints that will be marked as incorrect and wont be included in the acceptable supported list
	}{
		//Empty annotations should return empty hints
		{
			name:                   "Empty_Annotations",
			annotations:            map[string]string{},
			result:                 mapstr.M{},
			expectedIncorrectHints: 0,
		},

		// Scenarios being tested:
		// logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// logs/processors.add_fields must be nested mapstr.M under hints.logs
		// logs/json.keys_under_root must be a nested mapstr.M under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			name: "Logs_multiline_and_metrics",
			annotations: map[string]string{
				"co.elastic.logs/multiline.pattern":    "^test",
				"co.elastic.logs/json.keys_under_root": "true",
				"co.elastic.metrics/module":            "prometheus",
				"co.elastic.metrics/period":            "10s",
				"co.elastic.metrics.foobar/period":     "15s",
				"co.elastic.metrics.foobar1/period":    "15s",
				"not.to.include":                       "true",
			},
			result: mapstr.M{
				"logs": mapstr.M{
					"multiline": mapstr.M{
						"pattern": "^test",
					},
					"json": mapstr.M{
						"keys_under_root": "true",
					},
				},
				"metrics": mapstr.M{
					"module": "prometheus",
					"period": "15s",
				},
			},
			expectedIncorrectHints: 0,
		},
		// Scenarios being tested:
		// logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// metrics/metrics_path must be found in hints.metrics
		{
			name: "Logs_multiline_and_metrics_with_metrics_path",
			annotations: map[string]string{
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics/metrics_path":   "/metrics/prometheus",
				"co.elastic.metrics/username":       "user",
				"co.elastic.metrics/password":       "pass",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"co.elastic.hints/steam":            "stdout", // On purpose this added with typo
				"not.to.include":                    "true",
			},
			result: mapstr.M{
				"logs": mapstr.M{
					"multiline": mapstr.M{
						"pattern": "^test",
					},
				},
				"hints": mapstr.M{"steam": "stdout"},
				"metrics": mapstr.M{
					"module":       "prometheus",
					"period":       "15s",
					"metrics_path": "/metrics/prometheus",
					"username":     "user",
					"password":     "pass",
				},
			},
			expectedIncorrectHints: 1, // Due to co.elastic.hints/steam and not co.elastic.hints/stream
		},
		// Scenarios being tested:
		// logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			name: "Logs_multiline_and_metrics",
			annotations: map[string]string{
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			result: mapstr.M{
				"logs": mapstr.M{
					"multiline": mapstr.M{
						"pattern": "^test",
					},
				},
				"metrics": mapstr.M{
					"module": "prometheus",
					"period": "15s",
				},
			},
			expectedIncorrectHints: 0,
		},
		// Scenarios being tested:
		// have co.elastic.logs/disable set to false.
		// logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			name: "Logs_disabled_false_and_metrics",
			annotations: map[string]string{
				"co.elastic.logs/disable":           "false",
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			result: mapstr.M{
				"logs": mapstr.M{
					"multiline": mapstr.M{
						"pattern": "^test",
					},
					"disable": "false",
				},
				"metrics": mapstr.M{
					"module": "prometheus",
					"period": "15s",
				},
			},
			expectedIncorrectHints: 0,
		},
		// Scenarios being tested:
		// have co.elastic.logs/disable set to true.
		// logs/multiline.pattern must be a nested mapstr.M under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			name: "Logs_disabled_true_and_metrics",
			annotations: map[string]string{
				"co.elastic.logs/disable":           "true",
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			result: mapstr.M{
				"logs": mapstr.M{
					"multiline": mapstr.M{
						"pattern": "^test",
					},
					"disable": "true",
				},
				"metrics": mapstr.M{
					"module": "prometheus",
					"period": "15s",
				},
			},
			expectedIncorrectHints: 0,
		},
	}

	for _, test := range tests {
		annMap := mapstr.M{}
		for k, v := range test.annotations {
			_, err := annMap.Put(k, v)
			if err != nil {
				continue
			}
		}
		generateHints, incorrectHints := GenerateHints(annMap, "foobar", "co.elastic", allSupportedHints)
		assert.Equal(t, test.expectedIncorrectHints, len(incorrectHints)) // We validate how many incorrect hints are provided per test case.
		assert.Equal(t, test.result, generateHints)
	}
}
func TestGetHintsAsList(t *testing.T) {
	tests := []struct {
		input   mapstr.M
		output  []mapstr.M
		message string
	}{
		{
			input: mapstr.M{
				"metrics": mapstr.M{
					"module": "prometheus",
					"period": "15s",
				},
			},
			output: []mapstr.M{
				{
					"module": "prometheus",
					"period": "15s",
				},
			},
			message: "Single hint should return a single set of configs",
		},
		{
			input: mapstr.M{
				"metrics": mapstr.M{
					"1": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
				},
			},
			output: []mapstr.M{
				{
					"module": "prometheus",
					"period": "15s",
				},
			},
			message: "Single hint with numeric prefix should return a single set of configs",
		},
		{
			input: mapstr.M{
				"metrics": mapstr.M{
					"1": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
					"2": mapstr.M{
						"module": "dropwizard",
						"period": "20s",
					},
				},
			},
			output: []mapstr.M{
				{
					"module": "prometheus",
					"period": "15s",
				},
				{
					"module": "dropwizard",
					"period": "20s",
				},
			},
			message: "Multiple hints with numeric prefix should return configs in numeric ordering",
		},
		{
			input: mapstr.M{
				"metrics": mapstr.M{
					"1": mapstr.M{
						"module": "prometheus",
						"period": "15s",
					},
					"module": "dropwizard",
					"period": "20s",
				},
			},
			output: []mapstr.M{
				{
					"module": "prometheus",
					"period": "15s",
				},
				{
					"module": "dropwizard",
					"period": "20s",
				},
			},
			message: "Multiple hints with numeric prefix and default should return configs with defaults at the last",
		},
	}

	for _, test := range tests {
		t.Run(test.message, func(t *testing.T) {
			assert.Equal(t, test.output, GetHintsAsList(test.input, "metrics"))
		})
	}
}
