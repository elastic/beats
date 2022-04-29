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

package actions

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestCopyFields(t *testing.T) {
	log := logp.NewLogger("copy_fields_test")
	var tests = map[string]struct {
		FromTo   fromTo
		Input    mapstr.M
		Expected mapstr.M
	}{
		"copy string from message to message_copied": {
			FromTo: fromTo{
				From: "message",
				To:   "message_copied",
			},
			Input: mapstr.M{
				"message": "please copy this line",
			},
			Expected: mapstr.M{
				"message":        "please copy this line",
				"message_copied": "please copy this line",
			},
		},
		"copy string from nested key nested.message to top level field message_copied": {
			FromTo: fromTo{
				From: "nested.message",
				To:   "message_copied",
			},
			Input: mapstr.M{
				"nested": mapstr.M{
					"message": "please copy this line",
				},
			},
			Expected: mapstr.M{
				"nested": mapstr.M{
					"message": "please copy this line",
				},
				"message_copied": "please copy this line",
			},
		},
		"copy string from fieldname with dot to message_copied": {
			FromTo: fromTo{
				From: "dotted.message",
				To:   "message_copied",
			},
			Input: mapstr.M{
				"dotted.message": "please copy this line",
			},
			Expected: mapstr.M{
				"dotted.message": "please copy this line",
				"message_copied": "please copy this line",
			},
		},
		"copy number from fieldname with dot to dotted message.copied": {
			FromTo: fromTo{
				From: "message.original",
				To:   "message.copied",
			},
			Input: mapstr.M{
				"message.original": 42,
			},
			Expected: mapstr.M{
				"message.original": 42,
				"message": mapstr.M{
					"copied": 42,
				},
			},
		},
		"copy number from hierarchical message.original to top level message which fails": {
			FromTo: fromTo{
				From: "message.original",
				To:   "message",
			},
			Input: mapstr.M{
				"message": mapstr.M{
					"original": 42,
				},
			},
			Expected: mapstr.M{
				"message": mapstr.M{
					"original": 42,
				},
			},
		},
		"copy number from hierarchical message.original to top level message": {
			FromTo: fromTo{
				From: "message.original",
				To:   "message",
			},
			Input: mapstr.M{
				"message.original": 42,
			},
			Expected: mapstr.M{
				"message.original": 42,
				"message":          42,
			},
		},
		"copy map from nested key message.original to top level field message_copied": {
			FromTo: fromTo{
				From: "message.original",
				To:   "message_copied",
			},
			Input: mapstr.M{
				"message": mapstr.M{
					"original": mapstr.M{
						"original": "original",
					},
				},
			},
			Expected: mapstr.M{
				"message": mapstr.M{
					"original": mapstr.M{
						"original": "original",
					},
				},
				"message_copied": mapstr.M{
					"original": "original",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := copyFields{
				copyFieldsConfig{
					Fields: []fromTo{
						test.FromTo,
					},
				},
				log,
			}

			event := &beat.Event{
				Fields: test.Input,
			}

			newEvent, err := p.Run(event)
			assert.NoError(t, err)

			assert.Equal(t, test.Expected, newEvent.Fields)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		p := copyFields{
			copyFieldsConfig{
				Fields: []fromTo{
					{
						From: "@metadata.message",
						To:   "@metadata.message_copied",
					},
				},
			},
			log,
		}

		event := &beat.Event{
			Meta: mapstr.M{
				"message": "please copy this line",
			},
		}

		expMeta := mapstr.M{
			"message":        "please copy this line",
			"message_copied": "please copy this line",
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
