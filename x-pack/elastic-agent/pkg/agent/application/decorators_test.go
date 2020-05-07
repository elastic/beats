// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

func TestInjectV2Templates(t *testing.T) {
	const outputGroup = "default"
	t.Run("inject parameters on elasticsearch output", func(t *testing.T) {

		config := map[string]interface{}{
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "elasticsearch",
					"username": "foo",
					"password": "secret",
				},
			},
			"datasources": []map[string]interface{}{
				map[string]interface{}{
					"inputs": []map[string]interface{}{
						map[string]interface{}{
							"type": "log",
							"streams": []map[string]interface{}{
								map[string]interface{}{"paths": "/xxxx"},
							},
							"processors": []interface{}{
								map[string]interface{}{
									"dissect": map[string]interface{}{
										"tokenizer": "---",
									},
								},
							},
						},
					},
				},
				map[string]interface{}{
					"inputs": []map[string]interface{}{
						map[string]interface{}{
							"type": "system/metrics",
							"streams": []map[string]interface{}{
								map[string]interface{}{
									"id":      "system/metrics-system.core",
									"enabled": true,
									"dataset": "system.core",
									"period":  "10s",
									"metrics": []string{"percentages"},
								},
							},
						},
					},
					"use_output": "default",
				},
			},
		}

		ast, err := transpiler.NewAST(config)
		if err != nil {
			t.Fatal(err)
		}

		programsToRun, err := program.Programs(ast)
		if err != nil {
			t.Fatal(err)
		}

		programsWithOutput, err := injectPreferV2Template(outputGroup, ast, programsToRun[outputGroup])
		require.NoError(t, err)

		assert.Equal(t, len(programsToRun[outputGroup]), len(programsWithOutput))

		for _, program := range programsWithOutput {
			m, err := program.Config.Map()
			require.NoError(t, err)

			want := map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"username": "foo",
					"password": "secret",
					"parameters": map[string]interface{}{
						"prefer_v2_templates": true,
					},
				},
			}

			got, _ := m["output"]
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		}
	})

	t.Run("dont do anything on logstash output", func(t *testing.T) {
		config := map[string]interface{}{
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"type":     "logstash",
					"username": "foo",
					"password": "secret",
				},
			},
			"datasources": []map[string]interface{}{
				map[string]interface{}{
					"inputs": []map[string]interface{}{
						map[string]interface{}{
							"type": "log",
							"streams": []map[string]interface{}{
								map[string]interface{}{"paths": "/xxxx"},
							},
							"processors": []interface{}{
								map[string]interface{}{
									"dissect": map[string]interface{}{
										"tokenizer": "---",
									},
								},
							},
						},
					},
				},
				map[string]interface{}{
					"inputs": []map[string]interface{}{
						map[string]interface{}{
							"type": "system/metrics",
							"streams": []map[string]interface{}{
								map[string]interface{}{
									"id":      "system/metrics-system.core",
									"enabled": true,
									"dataset": "system.core",
									"period":  "10s",
									"metrics": []string{"percentages"},
								},
							},
						},
					},
					"use_output": "default",
				},
			},
		}

		ast, err := transpiler.NewAST(config)
		if err != nil {
			t.Fatal(err)
		}

		programsToRun, err := program.Programs(ast)
		if err != nil {
			t.Fatal(err)
		}

		programsWithOutput, err := injectPreferV2Template(outputGroup, ast, programsToRun[outputGroup])
		require.NoError(t, err)

		assert.Equal(t, len(programsToRun[outputGroup]), len(programsWithOutput))

		for _, program := range programsWithOutput {
			m, err := program.Config.Map()
			require.NoError(t, err)

			want := map[string]interface{}{
				"logstash": map[string]interface{}{
					"username": "foo",
					"password": "secret",
				},
			}

			got, _ := m["output"]
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		}
	})
}
