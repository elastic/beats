// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

func TestInjectESOutputAPIKey(t *testing.T) {
	decodedAPIKey := "hello:world"
	APIKey := base64.StdEncoding.EncodeToString([]byte(decodedAPIKey))

	tests := map[string]struct {
		config   []program.Program
		expected []program.Program
	}{
		"Single program with elasticsearch output": {
			config: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts": "xxx",
							},
						},
					),
				},
			},
			expected: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts":   "xxx",
								"api_key": decodedAPIKey,
							},
						},
					),
				},
			},
		},
		"Multiples programs with elasticsearch output": {
			config: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts": "xxx",
							},
						},
					),
				},
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"modules": []map[string]interface{}{
								map[string]interface{}{
									"module": "nginx",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts": "xxx",
							},
						},
					),
				},
			},
			expected: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts":   "xxx",
								"api_key": decodedAPIKey,
							},
						},
					),
				},
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"modules": []map[string]interface{}{
								map[string]interface{}{
									"module": "nginx",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"api_key": decodedAPIKey,
								"hosts":   "xxx",
							},
						},
					),
				},
			},
		},
		"Single program with elasticsearch output with an existing api key": {
			config: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts":   "xxx",
								"api_key": "another:apikey",
							},
						},
					),
				},
			},
			expected: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.elasticsearch": map[string]interface{}{
								"hosts":   "xxx",
								"api_key": "another:apikey",
							},
						},
					),
				},
			},
		},
		"Single program with Logstash output": {
			config: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.logstash": map[string]interface{}{
								"hosts": "xxx",
							},
						},
					),
				},
			},
			expected: []program.Program{
				program.Program{
					Config: transpiler.MustNewAST(
						map[string]interface{}{
							"inputs": []map[string]interface{}{
								map[string]interface{}{
									"type": "log",
									"path": "/var/hello.log",
								},
							},
							"output.logstash": map[string]interface{}{
								"hosts": "xxx",
							},
						},
					),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			decorate, err := injectESOutputAPIKey(APIKey)
			require.NoError(t, err)

			programs, err := decorate("", nil, test.config)
			require.NoError(t, err)

			assert.Equal(t, len(test.expected), len(programs))
			for idx, expected := range test.expected {
				if !assert.True(t, cmp.Equal(expected.Configuration(), programs[idx].Configuration())) {
					diff := cmp.Diff(expected, programs[idx])
					if diff != "" {
						t.Errorf("%s-%d mismatch (-want +got):\n%s", name, idx, diff)
					}
				}
			}
		})
	}
}

// "Program without an elasticsearch output defined"
