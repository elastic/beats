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

package urlparse

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestURLParse(t *testing.T) {
	var testCases = []struct {
		description string
		config      urlParseConfig
		Input       common.MapStr
		Output      common.MapStr
		error       bool
	}{
		{
			description: "simple field urlparse",
			config: urlParseConfig{
				Fields: []fromTo{{
					From: "field1", To: "field2",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "https://hello.world.com:8999/index.html?hello=world#main",
			},
			Output: common.MapStr{
				"field1": "https://hello.world.com:8999/index.html?hello=world#main",
				"field2": common.MapStr{
					"scheme":    "https",
					"opaque":    "",
					"hostname":  "hello.world.com",
					"port":      "8999",
					"path":      "/index.html",
					"raw_path":  "",
					"raw_query": "hello=world",
					"fragment":  "main",
				},
			},
			error: false,
		},
		{
			description: "simple multiple fields urlparse",
			config: urlParseConfig{
				Fields: []fromTo{
					{From: "field1", To: "field2"},
					{From: "field3", To: "field4"},
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "https://hello.world.com/youjiantaolovewangping",
				"field3": "https://hello.world.com/bonsailovesarah",
			},
			Output: common.MapStr{
				"field1": "https://hello.world.com/youjiantaolovewangping",
				"field2": common.MapStr{
					"scheme":    "https",
					"opaque":    "",
					"hostname":  "hello.world.com",
					"port":      "",
					"path":      "/youjiantaolovewangping",
					"raw_path":  "",
					"raw_query": "",
					"fragment":  "",
				},
				"field3": "https://hello.world.com/bonsailovesarah",
				"field4": common.MapStr{
					"scheme":    "https",
					"opaque":    "",
					"hostname":  "hello.world.com",
					"port":      "",
					"path":      "/bonsailovesarah",
					"raw_path":  "",
					"raw_query": "",
					"fragment":  "",
				},
			},
			error: false,
		},
		{
			description: "simple field urlparse To empty",
			config: urlParseConfig{
				Fields: []fromTo{{
					From: "field1", To: "",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "https://hello.world.com",
			},
			Output: common.MapStr{
				"field1": common.MapStr{
					"scheme":    "https",
					"opaque":    "",
					"hostname":  "hello.world.com",
					"port":      "",
					"path":      "",
					"raw_path":  "",
					"raw_query": "",
					"fragment":  "",
				},
			},
			error: false,
		},
		{
			description: "simple field urlparse from and to equals",
			config: urlParseConfig{
				Fields: []fromTo{{
					From: "field1", To: "field1",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "https://hello.world.com",
			},
			Output: common.MapStr{
				"field1": common.MapStr{
					"scheme":    "https",
					"opaque":    "",
					"hostname":  "hello.world.com",
					"port":      "",
					"path":      "",
					"raw_path":  "",
					"raw_query": "",
					"fragment":  "",
				},
			},
			error: false,
		},
		{
			description: "simple field urlparse with opaque",
			config: urlParseConfig{
				Fields: []fromTo{{
					From: "field1", To: "field1",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "https:%2f%2fhello.world.com",
			},
			Output: common.MapStr{
				"field1": common.MapStr{
					"scheme":    "https",
					"opaque":    "%2f%2fhello.world.com",
					"hostname":  "",
					"port":      "",
					"path":      "",
					"raw_path":  "",
					"raw_query": "",
					"fragment":  "",
				},
			},
			error: false,
		},
		{
			description: "simple field urlparse with raw_path",
			config: urlParseConfig{
				Fields: []fromTo{{
					From: "field1", To: "field1",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "https://hello.world.com/%47%6f",
			},
			Output: common.MapStr{
				"field1": common.MapStr{
					"scheme":    "https",
					"opaque":    "",
					"hostname":  "hello.world.com",
					"port":      "",
					"path":      "/Go",
					"raw_path":  "/%47%6f",
					"raw_query": "",
					"fragment":  "",
				},
			},
			error: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			f := &urlParse{
				log:    logp.NewLogger("urlparse"),
				config: test.config,
			}

			event := &beat.Event{
				Fields: test.Input,
			}

			newEvent, err := f.Run(event)
			if !test.error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, test.Output, newEvent.Fields)

		})
	}

}
