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
	"math"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestJsonCodec(t *testing.T) {
	type testCase struct {
		config   Config
		ts       time.Time
		in       mapstr.M
		expected string
	}

	cases := map[string]testCase{
		"default json": testCase{
			config:   defaultConfig,
			in:       mapstr.M{"msg": "message"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"_doc","version":"1.2.3"},"msg":"message"}`,
		},
		"pretty enabled": testCase{
			config: Config{Pretty: true},
			in:     mapstr.M{"msg": "message"},
			expected: `{
  "@timestamp": "0001-01-01T00:00:00.000Z",
  "@metadata": {
    "beat": "test",
    "type": "_doc",
    "version": "1.2.3"
  },
  "msg": "message"
}`,
		},
		"html escaping enabled": {
			config:   Config{EscapeHTML: true},
			in:       mapstr.M{"msg": "<hello>world</hello>"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"_doc","version":"1.2.3"},"msg":"\u003chello\u003eworld\u003c/hello\u003e"}`,
		},
		"html escaping disabled": {
			config:   Config{EscapeHTML: false},
			in:       mapstr.M{"msg": "<hello>world</hello>"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"_doc","version":"1.2.3"},"msg":"<hello>world</hello>"}`,
		},
		"UTC timezone offset": {
			config:   Config{LocalTime: true},
			in:       mapstr.M{"msg": "message"},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000+00:00","@metadata":{"beat":"test","type":"_doc","version":"1.2.3"},"msg":"message"}`,
		},
		"PST timezone offset": {
			config:   Config{LocalTime: true},
			ts:       time.Time{}.In(time.FixedZone("PST", -8*60*60)),
			in:       mapstr.M{"msg": "message"},
			expected: `{"@timestamp":"0000-12-31T16:00:00.000-08:00","@metadata":{"beat":"test","type":"_doc","version":"1.2.3"},"msg":"message"}`,
		},
		"float undefined values": {
			in:       mapstr.M{"nan": math.NaN()},
			expected: `{"@timestamp":"0001-01-01T00:00:00.000Z","@metadata":{"beat":"test","type":"_doc","version":"1.2.3"},"nan":null}`,
		},
	}

	for name, test := range cases {
		cfg, ts, fields, expected := test.config, test.ts, test.in, test.expected

		t.Run(name, func(t *testing.T) {
			codec := New("1.2.3", cfg)
			actual, err := codec.Encode("test", &beat.Event{Fields: fields, Timestamp: ts})

			if err != nil {
				t.Errorf("Error during event write %v", err)
			} else if string(actual) != expected {
				t.Errorf("Expected value (%s) does not equal with output (%s)", expected, actual)
			}
		})
	}
}
