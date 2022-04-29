// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfigBlacklistSettingsUnpack(t *testing.T) {
	tests := []struct {
		name     string
		config   *conf.C
		error    bool
		expected ConfigBlacklistSettings
	}{
		{
			name: "Simple config",
			config: conf.MustNewConfigFrom(map[string]interface{}{
				"foo": "bar",
			}),
			expected: ConfigBlacklistSettings{
				Patterns: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			name:   "Wrong config",
			config: conf.MustNewConfigFrom([]string{"a", "b"}),
			error:  true,
		},
		{
			name: "Tree config",
			config: conf.MustNewConfigFrom(map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			}),
			expected: ConfigBlacklistSettings{
				Patterns: map[string]string{
					"foo.bar": "baz",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result ConfigBlacklistSettings
			err := test.config.Unpack(&result)
			if test.error {
				assert.Error(t, err)
			}
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestConfigBlacklist(t *testing.T) {
	tests := []struct {
		name        string
		patterns    map[string]string
		blocks      ConfigBlocks
		blacklisted bool
	}{
		{
			name:        "No patterns",
			blacklisted: false,
			blocks: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "output",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"output": "console",
							},
						},
					},
				},
			},
		},
		{
			name:        "Blacklisted dict key",
			blacklisted: true,
			patterns: map[string]string{
				"output": "^console$",
			},
			blocks: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "output",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"console": map[string]interface{}{
									"pretty": "true",
								},
							},
						},
						{
							Raw: map[string]interface{}{
								"elasticsearch": map[string]interface{}{
									"host": "localhost",
								},
							},
						},
					},
				},
			},
		},
		{
			name:        "Blacklisted value key",
			blacklisted: true,
			patterns: map[string]string{
				"metricbeat.modules.module": "k.{8}s",
			},
			blocks: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"module": "kubernetes",
								"hosts":  "localhost:10255",
							},
						},
					},
				},
			},
		},
		{
			name:        "Blacklisted value in a list",
			blacklisted: true,
			patterns: map[string]string{
				"metricbeat.modules.metricsets": "event",
			},
			blocks: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"module": "kubernetes",
								"metricsets": []string{
									"event",
									"default",
								},
							},
						},
						{
							Raw: map[string]interface{}{
								"module": "kubernetes",
								"metricsets": []string{
									"default",
								},
							},
						},
					},
				},
			},
		},
		{
			name:        "Blacklisted value in a deep list",
			blacklisted: true,
			patterns: map[string]string{
				"filebeat.inputs.containers.ids": "1ffeb0dbd13",
			},
			blocks: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"module": "kubernetes",
								"metricsets": []string{
									"event",
									"default",
								},
							},
						},
					},
				},
				ConfigBlocksWithType{
					Type: "filebeat.inputs",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"type": "docker",
								"containers": map[string]interface{}{
									"ids": []string{
										"1ffeb0dbd13",
									},
								},
							},
						},
						{
							Raw: map[string]interface{}{
								"type": "docker",
								"containers": map[string]interface{}{
									"ids": []string{
										"256425931c2",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:        "Blacklisted dict key in a list",
			blacklisted: true,
			patterns: map[string]string{
				"list.of.elements":            "forbidden",
				"list.of.elements.disallowed": "yes",
			},
			blocks: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "list",
					Blocks: []*ConfigBlock{
						{
							Raw: map[string]interface{}{
								"of": map[string]interface{}{
									"elements": []interface{}{
										map[string]interface{}{
											"forbidden": "yes",
										},
									},
								},
							},
						},
						{
							Raw: map[string]interface{}{
								"of": map[string]interface{}{
									"elements": []interface{}{
										map[string]interface{}{
											"allowed": "yes",
										},
									},
								},
							},
						},
						{
							Raw: map[string]interface{}{
								"of": map[string]interface{}{
									"elements": []interface{}{
										map[string]interface{}{
											"disallowed": "yes",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := ConfigBlacklistSettings{
				Patterns: test.patterns,
			}
			bl, err := NewConfigBlacklist(cfg)
			if err != nil {
				t.Fatal(err)
			}

			errs := bl.Detect(test.blocks)
			if test.blacklisted {
				assert.NotNil(t, errs)
			} else {
				assert.Nil(t, errs)
			}
		})
	}
}
