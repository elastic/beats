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

package json

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestJsonCodec(t *testing.T) {
	type testCase struct {
		config   config
		in       common.MapStr
		expected string
	}

	cases := map[string]testCase{
		"default json": testCase{
			config:   defaultConfig,
			in:       common.MapStr{"msg": "message"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"doc","version":"1.2.3"},"msg":"message"}`,
		},
		"pretty enabled": testCase{
			config: config{Pretty: true},
			in:     common.MapStr{"msg": "message"},
			expected: `{
  "@timestamp": "0001-01-01T00:00:00.000Z",
  "@metadata": {
    "beat": "test",
    "type": "doc",
    "version": "1.2.3"
  },
  "msg": "message"
}`,
		},
		"html escaping enabled": testCase{
			config:   config{EscapeHTML: true},
			in:       common.MapStr{"msg": "<hello>world</hello>"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"doc","version":"1.2.3"},"msg":"\u003chello\u003eworld\u003c/hello\u003e"}`,
		},
		"html escaping disabled": testCase{
			config:   config{EscapeHTML: false},
			in:       common.MapStr{"msg": "<hello>world</hello>"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"doc","version":"1.2.3"},"msg":"<hello>world</hello>"}`,
		},
	}

	for name, test := range cases {
		cfg, fields, expected := test.config, test.in, test.expected

		t.Run(name, func(t *testing.T) {
			codec := New(cfg.Pretty, cfg.EscapeHTML, "1.2.3")
			actual, err := codec.Encode("test", &beat.Event{Fields: fields})

			if err != nil {
				t.Errorf("Error during event write %v", err)
			} else if string(actual) != expected {
				t.Errorf("Expected value (%s) does not equal with output (%s)", expected, actual)
			}
		})
	}
}
