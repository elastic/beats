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
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type node map[string]interface{}

func TestSelector(t *testing.T) {
	useLowerCase := func(s Settings) Settings {
		return s.WithSelectorCase(SelectorLowerCase)
	}

	tests := map[string]struct {
		config   string
		event    mapstr.M
		want     string
		settings func(Settings) Settings
	}{
		"constant key": {
			config: `key: value`,
			event:  mapstr.M{},
			want:   "value",
		},
		"lowercase constant key": {
			config:   `key: VaLuE`,
			event:    mapstr.M{},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase constant key by default": {
			config: `key: VaLuE`,
			event:  mapstr.M{},
			want:   "VaLuE",
		},
		"format string key": {
			config: `key: '%{[key]}'`,
			event:  mapstr.M{"key": "value"},
			want:   "value",
		},
		"lowercase format string key": {
			config:   `key: '%{[key]}'`,
			event:    mapstr.M{"key": "VaLuE"},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase format string by default": {
			config: `key: '%{[key]}'`,
			event:  mapstr.M{"key": "VaLuE"},
			want:   "VaLuE",
		},
		"key with empty keys": {
			config: `{key: value, keys: }`,
			event:  mapstr.M{},
			want:   "value",
		},
		"lowercase key with empty keys": {
			config:   `{key: vAlUe, keys: }`,
			event:    mapstr.M{},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase key with empty keys by default": {
			config: `{key: vAlUe, keys: }`,
			event:  mapstr.M{},
			want:   "vAlUe",
		},
		"constant in multi key": {
			config: `keys: [key: 'value']`,
			event:  mapstr.M{},
			want:   "value",
		},
		"format string in multi key": {
			config: `keys: [key: '%{[key]}']`,
			event:  mapstr.M{"key": "value"},
			want:   "value",
		},
		"missing format string key with default in rule": {
			config: `keys:
			        - key: '%{[key]}'
			          default: value`,
			event: mapstr.M{},
			want:  "value",
		},
		"lowercase missing format string key with default in rule": {
			config: `keys:
			        - key: '%{[key]}'
			          default: vAlUe`,
			event:    mapstr.M{},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase missing format string key with default in rule": {
			config: `keys:
			        - key: '%{[key]}'
			          default: vAlUe`,
			event: mapstr.M{},
			want:  "vAlUe",
		},
		"empty format string key with default in rule": {
			config: `keys:
						        - key: '%{[key]}'
						          default: value`,
			event: mapstr.M{"key": ""},
			want:  "value",
		},
		"lowercase empty format string key with default in rule": {
			config: `keys:
						        - key: '%{[key]}'
						          default: vAluE`,
			event:    mapstr.M{"key": ""},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase empty format string key with default in rule": {
			config: `keys:
						        - key: '%{[key]}'
						          default: vAluE`,
			event: mapstr.M{"key": ""},
			want:  "vAluE",
		},
		"missing format string key with constant in next rule": {
			config: `keys:
						        - key: '%{[key]}'
						        - key: value`,
			event: mapstr.M{},
			want:  "value",
		},
		"missing format string key with constant in top-level rule": {
			config: `{ key: value, keys: [key: '%{[key]}']}`,
			event:  mapstr.M{},
			want:   "value",
		},
		"apply mapping": {
			config: `keys:
						       - key: '%{[key]}'
						         mappings:
						           v: value`,
			event: mapstr.M{"key": "v"},
			want:  "value",
		},
		"lowercase applied mapping": {
			config: `keys:
						       - key: '%{[key]}'
						         mappings:
						           v: vAlUe`,
			event:    mapstr.M{"key": "v"},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase applied mapping": {
			config: `keys:
						       - key: '%{[key]}'
						         mappings:
						           v: vAlUe`,
			event: mapstr.M{"key": "v"},
			want:  "vAlUe",
		},
		"apply mapping with default on empty key": {
			config: `keys:
						       - key: '%{[key]}'
						         default: value
						         mappings:
						           v: 'v'`,
			event: mapstr.M{"key": ""},
			want:  "value",
		},
		"lowercase apply mapping with default on empty key": {
			config: `keys:
						       - key: '%{[key]}'
						         default: vAluE
						         mappings:
						           v: 'v'`,
			event:    mapstr.M{"key": ""},
			want:     "value",
			settings: useLowerCase,
		},
		"do not lowercase apply mapping with default on empty key": {
			config: `keys:
						       - key: '%{[key]}'
						         default: vAluE
						         mappings:
						           v: 'v'`,
			event: mapstr.M{"key": ""},
			want:  "vAluE",
		},
		"apply mapping with default on empty lookup": {
			config: `keys:
			       - key: '%{[key]}'
			         default: value
			         mappings:
			           v: ''`,
			event: mapstr.M{"key": "v"},
			want:  "value",
		},
		"apply mapping without match": {
			config: `keys:
						       - key: '%{[key]}'
						         mappings:
						           v: ''
						       - key: value`,
			event: mapstr.M{"key": "x"},
			want:  "value",
		},
		"mapping with constant key": {
			config: `keys:
						       - key: k
						         mappings:
						           k: value`,
			event: mapstr.M{},
			want:  "value",
		},
		"mapping with missing constant key": {
			config: `keys:
						       - key: unknown
						         mappings: {k: wrong}
						       - key: value`,
			event: mapstr.M{},
			want:  "value",
		},
		"mapping with missing constant key, but default": {
			config: `keys:
						       - key: unknown
						         default: value
						         mappings: {k: wrong}`,
			event: mapstr.M{},
			want:  "value",
		},
		"matching condition": {
			config: `keys:
						       - key: value
						         when.equals.test: test`,
			event: mapstr.M{"test": "test"},
			want:  "value",
		},
		"failing condition": {
			config: `keys:
						       - key: wrong
						         when.equals.test: test
						       - key: value`,
			event: mapstr.M{"test": "x"},
			want:  "value",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			yaml := strings.Replace(test.config, "\t", "  ", -1)
			cfg, err := config.NewConfigWithYAML([]byte(yaml), "test")
			if err != nil {
				t.Fatalf("YAML parse error: %v\n%v", err, yaml)
			}

			settings := Settings{
				Key:              "key",
				MultiKey:         "keys",
				EnableSingleOnly: true,
				FailEmpty:        true,
			}
			if test.settings != nil {
				settings = test.settings(settings)
			}

			sel, err := BuildSelectorFromConfig(cfg, settings)
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

			assert.Equal(t, test.want, actual)
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
			cfg, err := config.NewConfigWithYAML([]byte(test.config), "test")
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
