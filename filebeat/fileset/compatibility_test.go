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

//go:build !integration
// +build !integration

package fileset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestAdaptPipelineForCompatibility(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 6.7.0",
			esVersion: common.MustNewVersion("6.6.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				}},
			isErrExpected: true,
		},
		{
			name:      "ES == 6.7.0",
			esVersion: common.MustNewVersion("6.7.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
							"ecs":   true,
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES >= 7.0.0",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				},
			},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content)
			}
		})
	}
}

func TestReplaceSetIgnoreEmptyValue(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 7.9.0",
			esVersion: common.MustNewVersion("7.8.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": true,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "rule.name",
							"value": "{{panw.panos.ruleset}}",
							"if":    "ctx?.panw?.panos?.ruleset != null",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES == 7.9.0",
			esVersion: common.MustNewVersion("7.9.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": true,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": true,
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES > 7.9.0",
			esVersion: common.MustNewVersion("8.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": true,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": true,
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "existing if",
			esVersion: common.MustNewVersion("7.7.7"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": true,
							"if":                 "ctx?.panw?.panos?.ruleset != null",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "rule.name",
							"value": "{{panw.panos.ruleset}}",
							"if":    "ctx?.panw?.panos?.ruleset != null",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "ignore_empty_value is false",
			esVersion: common.MustNewVersion("7.7.7"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"value":              "{{panw.panos.ruleset}}",
							"ignore_empty_value": false,
							"if":                 "ctx?.panw?.panos?.ruleset != null",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "rule.name",
							"value": "{{panw.panos.ruleset}}",
							"if":    "ctx?.panw?.panos?.ruleset != null",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "no value",
			esVersion: common.MustNewVersion("7.7.7"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":              "rule.name",
							"ignore_empty_value": false,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "rule.name",
						},
					},
				}},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestReplaceAppendAllowDuplicates(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 7.10.0: set to true",
			esVersion: common.MustNewVersion("7.9.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": true,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
							"value": "{{host.hostname}}",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES < 7.10.0: set to false",
			esVersion: common.MustNewVersion("7.9.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
							"value": "{{host.hostname}}",
							"if":    "ctx?.host?.hostname != null && ((ctx?.related?.hosts instanceof List && !ctx?.related?.hosts.contains(ctx?.host?.hostname)) || ctx?.related?.hosts != ctx?.host?.hostname)",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES == 7.10.0",
			esVersion: common.MustNewVersion("7.10.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES > 7.10.0",
			esVersion: common.MustNewVersion("8.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES < 7.10.0: existing if",
			esVersion: common.MustNewVersion("7.7.7"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
							"if":               "ctx?.host?.hostname != null",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
							"value": "{{host.hostname}}",
							"if":    "ctx?.host?.hostname != null && ((ctx?.related?.hosts instanceof List && !ctx?.related?.hosts.contains(ctx?.host?.hostname)) || ctx?.related?.hosts != ctx?.host?.hostname)",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "ES < 7.10.0: existing if with contains",
			esVersion: common.MustNewVersion("7.7.7"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
							"if":               "!ctx?.related?.hosts.contains(ctx?.host?.hostname)",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
							"value": "{{host.hostname}}",
							"if":    "!ctx?.related?.hosts.contains(ctx?.host?.hostname)",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "ES < 7.10.0: no value",
			esVersion: common.MustNewVersion("7.7.7"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"allow_duplicates": false,
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
						},
					},
				}},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestRemoveURIPartsProcessor(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 7.12.0",
			esVersion: common.MustNewVersion("7.11.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"uri_parts": map[string]interface{}{
							"field":        "test.url",
							"target_field": "url",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES == 7.12.0",
			esVersion: common.MustNewVersion("7.12.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"uri_parts": map[string]interface{}{
							"field":        "test.url",
							"target_field": "url",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"uri_parts": map[string]interface{}{
							"field":        "test.url",
							"target_field": "url",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "ES > 7.12.0",
			esVersion: common.MustNewVersion("8.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"uri_parts": map[string]interface{}{
							"field":        "test.url",
							"target_field": "url",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"uri_parts": map[string]interface{}{
							"field":        "test.url",
							"target_field": "url",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestRemoveNetworkDirectionProcessor(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 7.13.0",
			esVersion: common.MustNewVersion("7.12.34"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"network_direction": map[string]interface{}{
							"internal_networks": []string{
								"loopback",
								"private",
							},
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES == 7.13.0",
			esVersion: common.MustNewVersion("7.13.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"network_direction": map[string]interface{}{
							"internal_networks": []string{
								"loopback",
								"private",
							},
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"network_direction": map[string]interface{}{
							"internal_networks": []string{
								"loopback",
								"private",
							},
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "ES > 7.13.0",
			esVersion: common.MustNewVersion("8.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"network_direction": map[string]interface{}{
							"internal_networks": []string{
								"loopback",
								"private",
							},
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"network_direction": map[string]interface{}{
							"internal_networks": []string{
								"loopback",
								"private",
							},
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestReplaceConvertIPWithGrok(t *testing.T) {
	logp.TestingSetup()
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES >= 7.13.0: keep processor",
			esVersion: common.MustNewVersion("7.13.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"convert": map[string]interface{}{
							"field":        "foo",
							"target_field": "bar",
							"type":         "ip",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"convert": map[string]interface{}{
							"field":        "foo",
							"target_field": "bar",
							"type":         "ip",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES < 7.13.0: replace with grok",
			esVersion: common.MustNewVersion("7.12.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"convert": map[string]interface{}{
							"field":        "foo",
							"target_field": "bar",
							"type":         "ip",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"grok": map[string]interface{}{
							"field": "foo",
							"patterns": []string{
								"^%{IP:bar}$",
							},
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "implicit target",
			esVersion: common.MustNewVersion("7.9.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"convert": map[string]interface{}{
							"field": "foo",
							"type":  "ip",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"grok": map[string]interface{}{
							"field": "foo",
							"patterns": []string{
								"^%{IP:foo}$",
							},
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "missing field",
			esVersion: common.MustNewVersion("7.9.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"convert": map[string]interface{}{
							"type": "ip",
						},
					},
				},
			},
			isErrExpected: true,
		},
		{
			name:      "keep settings in grok",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"convert": map[string]interface{}{
							"field":          "foo",
							"target_field":   "bar",
							"type":           "ip",
							"ignore_missing": true,
							"description":    "foo bar",
							"if":             "condition",
							"ignore_failure": false,
							"tag":            "myTag",
							"on_failure": []interface{}{
								map[string]interface{}{
									"foo": map[string]interface{}{
										"baz": false,
									},
								},
								map[string]interface{}{
									"bar": map[string]interface{}{
										"baz": true,
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"grok": map[string]interface{}{
							"field": "foo",
							"patterns": []string{
								"^%{IP:bar}$",
							},
							"ignore_missing": true,
							"if":             "condition",
							"ignore_failure": false,
							"tag":            "myTag",
							"on_failure": []interface{}{
								map[string]interface{}{
									"foo": map[string]interface{}{
										"baz": false,
									},
								},
								map[string]interface{}{
									"bar": map[string]interface{}{
										"baz": true,
									},
								},
							},
						},
					},
				},
			},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestRemoveRegisteredDomainProcessor(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 7.13.0",
			esVersion: common.MustNewVersion("7.12.34"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
					map[string]interface{}{
						"registered_domain": map[string]interface{}{
							"field": "foo",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES == 7.13.0",
			esVersion: common.MustNewVersion("7.13.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"registered_domain": map[string]interface{}{
							"field": "foo",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"registered_domain": map[string]interface{}{
							"field": "foo",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			isErrExpected: false,
		},
		{
			name:      "ES > 7.13.0",
			esVersion: common.MustNewVersion("8.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"registered_domain": map[string]interface{}{
							"field": "foo",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"registered_domain": map[string]interface{}{
							"field": "foo",
						},
					},
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "test.field",
							"value": "testvalue",
						},
					},
				}},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestReplaceAlternativeFlowProcessors(t *testing.T) {
	logp.TestingSetup()
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "Replace in on_failure section",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}(nil),
				"on_failure": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
						},
					},
					map[string]interface{}{
						"community_id": map[string]interface{}{},
					},
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "error.message",
							"value": "something's wrong",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}(nil),
				"on_failure": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
							"value": "{{host.hostname}}",
							"if":    "ctx?.host?.hostname != null && ((ctx?.related?.hosts instanceof List && !ctx?.related?.hosts.contains(ctx?.host?.hostname)) || ctx?.related?.hosts != ctx?.host?.hostname)",
						},
					},
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "error.message",
							"value": "something's wrong",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "Replace in processor's on_failure",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"foo": map[string]interface{}{
							"bar": "baz",
							"on_failure": []interface{}{
								map[string]interface{}{
									"append": map[string]interface{}{
										"field":            "related.hosts",
										"value":            "{{host.hostname}}",
										"allow_duplicates": false,
									},
								},
								map[string]interface{}{
									"community_id": map[string]interface{}{},
								},
								map[string]interface{}{
									"append": map[string]interface{}{
										"field": "error.message",
										"value": "something's wrong",
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"foo": map[string]interface{}{
							"bar": "baz",
							"on_failure": []interface{}{
								map[string]interface{}{
									"append": map[string]interface{}{
										"field": "related.hosts",
										"value": "{{host.hostname}}",
										"if":    "ctx?.host?.hostname != null && ((ctx?.related?.hosts instanceof List && !ctx?.related?.hosts.contains(ctx?.host?.hostname)) || ctx?.related?.hosts != ctx?.host?.hostname)",
									},
								},
								map[string]interface{}{
									"append": map[string]interface{}{
										"field": "error.message",
										"value": "something's wrong",
									},
								},
							},
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "Remove empty on_failure key",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"foo": map[string]interface{}{
							"bar": "baz",
							"on_failure": []interface{}{
								map[string]interface{}{
									"community_id": map[string]interface{}{},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"foo": map[string]interface{}{
							"bar": "baz",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "process foreach processor",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field":            "related.hosts",
							"value":            "{{host.hostname}}",
							"allow_duplicates": false,
							"if":               "ctx?.host?.hostname != null",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"append": map[string]interface{}{
							"field": "related.hosts",
							"value": "{{host.hostname}}",
							"if":    "ctx?.host?.hostname != null && ((ctx?.related?.hosts instanceof List && !ctx?.related?.hosts.contains(ctx?.host?.hostname)) || ctx?.related?.hosts != ctx?.host?.hostname)",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "Remove leftover foreach processor",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"foreach": map[string]interface{}{
							"field": "foo",
							"processor": map[string]interface{}{
								"community_id": map[string]interface{}{},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}(nil),
			},
			isErrExpected: false,
		},
		{
			name:      "nested",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}(nil),
				"on_failure": []interface{}{
					map[string]interface{}{
						"foreach": map[string]interface{}{
							"field": "foo",
							"processor": map[string]interface{}{
								"append": map[string]interface{}{
									"field":            "related.hosts",
									"value":            "{{host.hostname}}",
									"allow_duplicates": false,
									"if":               "ctx?.host?.hostname != null",
									"on_failure": []interface{}{
										map[string]interface{}{
											"community_id": map[string]interface{}{},
										},
										map[string]interface{}{
											"append": map[string]interface{}{
												"field": "error.message",
												"value": "panic",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}(nil),
				"on_failure": []interface{}{
					map[string]interface{}{
						"foreach": map[string]interface{}{
							"field": "foo",
							"processor": map[string]interface{}{
								"append": map[string]interface{}{
									"field": "related.hosts",
									"value": "{{host.hostname}}",
									"if":    "ctx?.host?.hostname != null && ((ctx?.related?.hosts instanceof List && !ctx?.related?.hosts.contains(ctx?.host?.hostname)) || ctx?.related?.hosts != ctx?.host?.hostname)",
									"on_failure": []interface{}{
										map[string]interface{}{
											"append": map[string]interface{}{
												"field": "error.message",
												"value": "panic",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}

func TestRemoveDescription(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 7.9.0",
			esVersion: common.MustNewVersion("7.8.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":       "rule.name",
							"value":       "{{panw.panos.ruleset}}",
							"description": "This is a description",
						},
					},
					map[string]interface{}{
						"script": map[string]interface{}{
							"source":      "abcd",
							"lang":        "painless",
							"description": "This is a description",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field": "rule.name",
							"value": "{{panw.panos.ruleset}}",
						},
					},
					map[string]interface{}{
						"script": map[string]interface{}{
							"source": "abcd",
							"lang":   "painless",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES == 7.9.0",
			esVersion: common.MustNewVersion("7.9.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":       "rule.name",
							"value":       "{{panw.panos.ruleset}}",
							"description": "This is a description",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":       "rule.name",
							"value":       "{{panw.panos.ruleset}}",
							"description": "This is a description",
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES > 7.9.0",
			esVersion: common.MustNewVersion("8.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":       "rule.name",
							"value":       "{{panw.panos.ruleset}}",
							"description": "This is a description",
						},
					},
				}},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"set": map[string]interface{}{
							"field":       "rule.name",
							"value":       "{{panw.panos.ruleset}}",
							"description": "This is a description",
						},
					},
				},
			},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := AdaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}
