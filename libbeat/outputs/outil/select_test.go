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

package outil

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

type node map[string]interface{}

func TestSelector(t *testing.T) {
	tests := map[string]struct {
		config   string
		event    common.MapStr
		expected string
	}{
		"constant key": {
			`key: value`,
			common.MapStr{},
			"value",
		},
		"lowercase constant key": {
			`key: VaLuE`,
			common.MapStr{},
			"value",
		},
		"format string key": {
			`key: '%{[key]}'`,
			common.MapStr{"key": "value"},
			"value",
		},
		"lowercase format string key": {
			`key: '%{[key]}'`,
			common.MapStr{"key": "VaLuE"},
			"value",
		},
		"key with empty keys": {
			`{key: value, keys: }`,
			common.MapStr{},
			"value",
		},
		"lowercase key with empty keys": {
			`{key: vAlUe, keys: }`,
			common.MapStr{},
			"value",
		},
		"constant in multi key": {
			`keys: [key: 'value']`,
			common.MapStr{},
			"value",
		},
		"format string in multi key": {
			`keys: [key: '%{[key]}']`,
			common.MapStr{"key": "value"},
			"value",
		},
		"missing format string key with default in rule": {
			`keys:
			        - key: '%{[key]}'
			          default: value`,
			common.MapStr{},
			"value",
		},
		"lowercase missing format string key with default in rule": {
			`keys:
			        - key: '%{[key]}'
			          default: vAlUe`,
			common.MapStr{},
			"value",
		},
		"empty format string key with default in rule": {
			`keys:
						        - key: '%{[key]}'
						          default: value`,
			common.MapStr{"key": ""},
			"value",
		},
		"lowercase empty format string key with default in rule": {
			`keys:
						        - key: '%{[key]}'
						          default: vAluE`,
			common.MapStr{"key": ""},
			"value",
		},
		"missing format string key with constant in next rule": {
			`keys:
						        - key: '%{[key]}'
						        - key: value`,
			common.MapStr{},
			"value",
		},
		"missing format string key with constant in top-level rule": {
			`{ key: value, keys: [key: '%{[key]}']}`,
			common.MapStr{},
			"value",
		},
		"apply mapping": {
			`keys:
						       - key: '%{[key]}'
						         mappings:
						           v: value`,
			common.MapStr{"key": "v"},
			"value",
		},
		"lowercase applied mapping": {
			`keys:
						       - key: '%{[key]}'
						         mappings:
						           v: vAlUe`,
			common.MapStr{"key": "v"},
			"value",
		},
		"apply mapping with default on empty key": {
			`keys:
						       - key: '%{[key]}'
						         default: value
						         mappings:
						           v: 'v'`,
			common.MapStr{"key": ""},
			"value",
		},
		"lowercase apply mapping with default on empty key": {
			`keys:
						       - key: '%{[key]}'
						         default: vAluE
						         mappings:
						           v: 'v'`,
			common.MapStr{"key": ""},
			"value",
		},
		"apply mapping with default on empty lookup": {
			`keys:
			       - key: '%{[key]}'
			         default: value
			         mappings:
			           v: ''`,
			common.MapStr{"key": "v"},
			"value",
		},
		"apply mapping without match": {
			`keys:
						       - key: '%{[key]}'
						         mappings:
						           v: ''
						       - key: value`,
			common.MapStr{"key": "x"},
			"value",
		},
		"mapping with constant key": {
			`keys:
						       - key: k
						         mappings:
						           k: value`,
			common.MapStr{},
			"value",
		},
		"mapping with missing constant key": {
			`keys:
						       - key: unknown
						         mappings: {k: wrong}
						       - key: value`,
			common.MapStr{},
			"value",
		},
		"mapping with missing constant key, but default": {
			`keys:
						       - key: unknown
						         default: value
						         mappings: {k: wrong}`,
			common.MapStr{},
			"value",
		},
		"matching condition": {
			`keys:
						       - key: value
						         when.equals.test: test`,
			common.MapStr{"test": "test"},
			"value",
		},
		"failing condition": {
			`keys:
						       - key: wrong
						         when.equals.test: test
						       - key: value`,
			common.MapStr{"test": "x"},
			"value",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			yaml := strings.Replace(test.config, "\t", "  ", -1)
			cfg, err := common.NewConfigWithYAML([]byte(yaml), "test")
			if err != nil {
				t.Fatalf("YAML parse error: %v\n%v", err, yaml)
			}

			sel, err := BuildSelectorFromConfig(cfg, Settings{
				Key:              "key",
				MultiKey:         "keys",
				EnableSingleOnly: true,
				FailEmpty:        true,
			})
			if err != nil {
				t.Fatal(err)
			}

			event := beat.Event{
				Timestamp: time.Now(),
				Fields:    test.event,
			}
			actual, err := sel.Select(&event)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestSelectorInitFail(t *testing.T) {
	tests := map[string]struct {
		config string
	}{
		"keys missing": {
			`test: no key`,
		},
		"invalid keys type": {
			`keys: 5`,
		},
		"invaid keys element type": {
			`keys: [5]`,
		},
		"invalid key type": {
			`key: {}`,
		},
		"missing key in list": {
			`keys: [default: value]`,
		},
		"invalid key type in list": {
			`keys: [key: {}]`,
		},
		"fail on invalid format string": {
			`key: '%{[abc}'`,
		},
		"fail on invalid format string in list": {
			`keys: [key: '%{[abc}']`,
		},
		"default value type mismatch": {
			`keys: [{key: ok, default: {}}]`,
		},
		"mappings type mismatch": {
			`keys:
       - key: '%{[k]}'
         mappings: {v: {}}`,
		},
		"condition empty": {
			`keys:
       - key: value
         when:`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := common.NewConfigWithYAML([]byte(test.config), "test")
			if err != nil {
				t.Fatal(err)
			}

			_, err = BuildSelectorFromConfig(cfg, Settings{
				Key:              "key",
				MultiKey:         "keys",
				EnableSingleOnly: true,
				FailEmpty:        true,
			})

			assert.Error(t, err)
			t.Log(err)
		})

	}
}
