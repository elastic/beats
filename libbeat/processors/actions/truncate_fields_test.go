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

	"github.com/menderesk/beats/v7/libbeat/logp"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

func TestTruncateFields(t *testing.T) {
	log := logp.NewLogger("truncate_fields_test")
	var tests = map[string]struct {
		MaxBytes     int
		MaxChars     int
		Input        common.MapStr
		Output       common.MapStr
		ShouldError  bool
		TruncateFunc truncater
	}{
		"truncate bytes of too long string line": {
			MaxBytes: 3,
			Input: common.MapStr{
				"message": "too long line",
			},
			Output: common.MapStr{
				"message": "too",
				"log": common.MapStr{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"truncate bytes of too long byte line": {
			MaxBytes: 3,
			Input: common.MapStr{
				"message": []byte("too long line"),
			},
			Output: common.MapStr{
				"message": []byte("too"),
				"log": common.MapStr{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"do not truncate short string line": {
			MaxBytes: 15,
			Input: common.MapStr{
				"message": "shorter line",
			},
			Output: common.MapStr{
				"message": "shorter line",
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"do not truncate short byte line": {
			MaxBytes: 15,
			Input: common.MapStr{
				"message": []byte("shorter line"),
			},
			Output: common.MapStr{
				"message": []byte("shorter line"),
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"try to truncate integer and get error": {
			MaxBytes: 5,
			Input: common.MapStr{
				"message": 42,
			},
			Output: common.MapStr{
				"message": 42,
			},
			ShouldError:  true,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"do not truncate characters of short byte line": {
			MaxChars: 6,
			Input: common.MapStr{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			Output: common.MapStr{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateCharacters,
		},
		"do not truncate bytes of short byte line with multibyte runes": {
			MaxBytes: 6,
			Input: common.MapStr{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			Output: common.MapStr{
				"message": []byte("ez jó"), // this is good (hungarian)
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
		"truncate characters of too long byte line": {
			MaxChars: 10,
			Input: common.MapStr{
				"message": []byte("ez egy túl hosszú sor"), // this is a too long line (hungarian)
			},
			Output: common.MapStr{
				"message": []byte("ez egy túl"), // this is a too (hungarian)
				"log": common.MapStr{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateCharacters,
		},
		"truncate bytes of too long byte line with multibyte runes": {
			MaxBytes: 10,
			Input: common.MapStr{
				"message": []byte("ez egy túl hosszú sor"), // this is a too long line (hungarian)
			},
			Output: common.MapStr{
				"message": []byte("ez egy tú"), // this is a "to" (hungarian)
				"log": common.MapStr{
					"flags": []string{"truncated"},
				},
			},
			ShouldError:  false,
			TruncateFunc: (*truncateFields).truncateBytes,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := truncateFields{
				config: truncateFieldsConfig{
					Fields:      []string{"message"},
					MaxBytes:    test.MaxBytes,
					MaxChars:    test.MaxChars,
					FailOnError: true,
				},
				truncate: test.TruncateFunc,
				logger:   log,
			}

			event := &beat.Event{
				Fields: test.Input,
			}

			newEvent, err := p.Run(event)
			if test.ShouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.Output, newEvent.Fields)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		p := truncateFields{
			config: truncateFieldsConfig{
				Fields:      []string{"@metadata.message"},
				MaxBytes:    3,
				FailOnError: true,
			},
			truncate: (*truncateFields).truncateBytes,
			logger:   log,
		}

		event := &beat.Event{
			Meta: common.MapStr{
				"message": "too long line",
			},
			Fields: common.MapStr{},
		}

		expFields := common.MapStr{
			"log": common.MapStr{
				"flags": []string{"truncated"},
			},
		}

		expMeta := common.MapStr{
			"message": "too",
		}

		newEvent, err := p.Run(event)
		assert.NoError(t, err)

		assert.Equal(t, expFields, newEvent.Fields)
		assert.Equal(t, expMeta, newEvent.Meta)
	})
}
