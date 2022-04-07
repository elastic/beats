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

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestDecompressGzip(t *testing.T) {
	var testCases = []struct {
		description string
		config      decompressGzipFieldConfig
		input       common.MapStr
		output      common.MapStr
		error       bool
	}{
		{
			description: "bytes field gzip decompress",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field1", To: "field2",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			input: common.MapStr{
				"field1": []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0},
			},
			output: common.MapStr{
				"field1": []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0},
				"field2": "decompressed data",
			},
			error: false,
		},
		{
			description: "string field gzip decompress",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field1", To: "field2",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			input: common.MapStr{
				"field1": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0}),
			},
			output: common.MapStr{
				"field1": string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0}),
				"field2": "decompressed data",
			},
			error: false,
		},
		{
			description: "simple field gzip decompress in place",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field1", To: "field1",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			input: common.MapStr{
				"field1": []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0},
			},
			output: common.MapStr{
				"field1": "decompressed data",
			},
			error: false,
		},
		{
			description: "invalid data - fail on error",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field1", To: "field1",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			input: common.MapStr{
				"field1": "invalid gzipped data",
			},
			output: common.MapStr{
				"field1": "invalid gzipped data",
				"error": common.MapStr{
					"message": "Failed to decompress field in decompress_gzip_field processor: error decompressing field field1: gzip: invalid header",
				},
			},
			error: true,
		},
		{
			description: "invalid data - do not fail",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field1", To: "field2",
				},
				IgnoreMissing: false,
				FailOnError:   false,
			},
			input: common.MapStr{
				"field1": "invalid gzipped data",
			},
			output: common.MapStr{
				"field1": "invalid gzipped data",
			},
			error: false,
		},
		{
			description: "missing field - do not ignore it",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field2", To: "field3",
				},
				IgnoreMissing: false,
				FailOnError:   true,
			},
			input: common.MapStr{
				"field1": "my value",
			},
			output: common.MapStr{
				"field1": "my value",
				"error": common.MapStr{
					"message": "Failed to decompress field in decompress_gzip_field processor: could not fetch value for key: field2, Error: key not found",
				},
			},
			error: true,
		},
		{
			description: "missing field ignore",
			config: decompressGzipFieldConfig{
				Field: fromTo{
					From: "field2", To: "field3",
				},
				IgnoreMissing: true,
				FailOnError:   true,
			},
			input: common.MapStr{
				"field1": "my value",
			},
			output: common.MapStr{
				"field1": "my value",
			},
			error: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			f := &decompressGzipField{
				log:    logp.NewLogger("decompress_gzip_field"),
				config: test.config,
			}

			event := &beat.Event{
				Fields: test.input,
			}

			newEvent, err := f.Run(event)
			if !test.error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, test.output, newEvent.Fields)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		t.Parallel()

		event := &beat.Event{
			Fields: common.MapStr{
				"field1": []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0},
			},
			Meta: common.MapStr{},
		}

		expectedMeta := common.MapStr{
			"field": "decompressed data",
		}

		config := decompressGzipFieldConfig{
			Field: fromTo{
				From: "field1", To: "@metadata.field",
			},
			IgnoreMissing: false,
			FailOnError:   true,
		}

		f := &decompressGzipField{
			log:    logp.NewLogger("decompress_gzip_field"),
			config: config,
		}

		newEvent, err := f.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, expectedMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}
