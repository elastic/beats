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

package urldecode

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestURLDecode(t *testing.T) {
	var testCases = []struct {
		description string
		config      urlDecodeConfig
		Input       common.MapStr
		Output      common.MapStr
		error       bool
	}{
		{
			description: "simple field urldecode",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field1", To: "field2",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "correct%20data",
			},
			Output: common.MapStr{
				"field1": "correct%20data",
				"field2": "correct data",
			},
			error: false,
		},
		{
			description: "simple multiple fields urldecode",
			config: urlDecodeConfig{
				Fields: []fromTo{
					{From: "field1", To: "field2"},
					{From: "field3", To: "field4"},
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "correct%20field1",
				"field3": "correct%20field3",
			},
			Output: common.MapStr{
				"field1": "correct%20field1",
				"field2": "correct field1",
				"field3": "correct%20field3",
				"field4": "correct field3",
			},
			error: false,
		},
		{
			description: "simple field urldecode To empty",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field1", To: "",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "correct%20data",
			},
			Output: common.MapStr{
				"field1": "correct data",
			},
			error: false,
		},
		{
			description: "simple field urldecode from and to equals",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field1", To: "field1",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "correct%20data",
			},
			Output: common.MapStr{
				"field1": "correct data",
			},
			error: false,
		},
		{
			description: "simple field bad data - fail on error",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field1", To: "field1",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "Hello G%ünter",
			},
			Output: common.MapStr{
				"field1": "Hello G%ünter",
				"error": common.MapStr{
					"message": "failed to decode fields in urldecode processor: error trying to URL-decode Hello G%ünter: invalid URL escape \"%ü\"",
				},
			},
			error: true,
		},
		{
			description: "simple field bad data fail on error false",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field1", To: "field1",
				}},
				IgnoreMissing: false,
				FailOnError:   false,
			},
			Input: common.MapStr{
				"field1": "Hello G%ünter",
			},
			Output: common.MapStr{
				"field1": "Hello G%ünter",
			},
			error: false,
		},
		{
			description: "missing field",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field2", To: "field3",
				}},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "correct%20data",
			},
			Output: common.MapStr{
				"field1": "correct%20data",
				"error": common.MapStr{
					"message": "failed to decode fields in urldecode processor: could not fetch value for key: field2, Error: key not found",
				},
			},
			error: true,
		},
		{
			description: "missing field ignore",
			config: urlDecodeConfig{
				Fields: []fromTo{{
					From: "field2", To: "field3",
				}},
				IgnoreMissing: true,
				FailOnError:   true,
			},
			Input: common.MapStr{
				"field1": "correct%20data",
			},
			Output: common.MapStr{
				"field1": "correct%20data",
			},
			error: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			f := &urlDecode{
				log:    logp.NewLogger("urldecode"),
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

	t.Run("supports metadata as a target", func(t *testing.T) {
		t.Parallel()

		config := urlDecodeConfig{
			Fields: []fromTo{{
				From: "@metadata.field", To: "@metadata.target",
			}},
		}

		f := &urlDecode{
			log:    logp.NewLogger("urldecode"),
			config: config,
		}

		event := &beat.Event{
			Meta: common.MapStr{
				"field": "correct%20data",
			},
		}
		expMeta := common.MapStr{
			"field":  "correct%20data",
			"target": "correct data",
		}
		newEvent, err := f.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
