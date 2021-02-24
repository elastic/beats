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
			err := adaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
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
			err := adaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
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
			err := adaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
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
			err := adaptPipelineForCompatibility(*test.esVersion, "foo-pipeline", test.content, logp.NewLogger(logName))
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, test.content, test.name)
			}
		})
	}
}
