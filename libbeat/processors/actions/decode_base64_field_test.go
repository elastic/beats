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

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestDecodeBase64Run(t *testing.T) {
	var testCases = []struct {
		description string
		config      base64Config
		Input       mapstr.M
		Output      mapstr.M
		error       bool
	}{
		{
			description: "simple field base64 decode",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "field2",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
			Output: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
				"field2": "correct data",
			},
			error: false,
		},
		{
			description: "simple field base64 decode To empty",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
			Output: mapstr.M{
				"field1": "correct data",
			},
			error: false,
		},
		{
			description: "simple field base64 decode from and to equals",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "field1",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
			Output: mapstr.M{
				"field1": "correct data",
			},
			error: false,
		},
		{
			description: "simple field padded base64 decode from and to equals",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "field1",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "Y29ycmVjdCBwYWRkZWQgZGF0YQ==",
			},
			Output: mapstr.M{
				"field1": "correct padded data",
			},
			error: false,
		},
		{
			description: "simple field unpadded base64 decode from and to equals",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "field1",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "dW5wYWRkZWQgZGF0YQ",
			},
			Output: mapstr.M{
				"field1": "unpadded data",
			},
			error: false,
		},
		{
			description: "simple field bad data - fail on error",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "field1",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "bad data",
			},
			Output: mapstr.M{
				"field1": "bad data",
				"error": mapstr.M{
					"message": "failed to decode base64 fields in processor: error trying to decode bad data: illegal base64 data at input byte 3",
				},
			},
			error: true,
		},
		{
			description: "simple field bad data fail on error false",
			config: base64Config{
				Field: fromTo{
					From: "field1", To: "field2",
				},
				IgnoreMissing: false,
				FailOnError:   false,
			},
			Input: mapstr.M{
				"field1": "bad data",
			},
			Output: mapstr.M{
				"field1": "bad data",
			},
			error: false,
		},
		{
			description: "missing field",
			config: base64Config{
				Field: fromTo{
					From: "field2", To: "field3",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
			Output: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
				"error": mapstr.M{
					"message": "failed to decode base64 fields in processor: could not fetch base64 value for key: field2, Error: key not found",
				},
			},
			error: true,
		},
		{
			description: "missing field ignore",
			config: base64Config{
				Field: fromTo{
					From: "field2", To: "field3",
				},
				IgnoreMissing: true,
				FailOnError:   true,
			},
			Input: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
			Output: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
			error: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			f := &decodeBase64Field{
				log:    logp.NewLogger(processorName),
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

	t.Run("supports a metadata field", func(t *testing.T) {
		config := base64Config{
			Field: fromTo{
				From: "field1",
				To:   "@metadata.field",
			},
		}

		event := &beat.Event{
			Meta: mapstr.M{},
			Fields: mapstr.M{
				"field1": "Y29ycmVjdCBkYXRh",
			},
		}

		f := &decodeBase64Field{
			log:    logp.NewLogger(processorName),
			config: config,
		}

		expectedFields := mapstr.M{
			"field1": "Y29ycmVjdCBkYXRh",
		}
		expectedMeta := mapstr.M{
			"field": "correct data",
		}

		newEvent, err := f.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expectedFields, newEvent.Fields)
		assert.Equal(t, expectedMeta, newEvent.Meta)
	})
}
