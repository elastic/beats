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

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/cfgtype"
	"github.com/menderesk/beats/v7/libbeat/reader/readfile"
)

type inputParsersConfig struct {
	MaxBytes       cfgtype.ByteSize        `config:"max_bytes"`
	LineTerminator readfile.LineTerminator `config:"line_terminator"`
	Parsers        Config                  `config:",inline"`
}

func TestParsersExampleInline(t *testing.T) {
	tests := map[string]struct {
		lines            string
		parsers          map[string]interface{}
		expectedMessages []string
	}{
		"multiline docker logs parser": {
			lines: `{"log":"[log] The following are log messages\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"[log] This one is\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":" on multiple\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":" lines","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"[log] In total there should be 3 events\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
`,
			parsers: map[string]interface{}{
				"max_bytes":       1024,
				"line_terminator": "auto",
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"keys_under_root": true,
							"message_key":     "log",
						},
					},
					map[string]interface{}{
						"multiline": map[string]interface{}{
							"match":   "after",
							"negate":  true,
							"pattern": "^\\[log\\]",
						},
					},
				},
			},
			expectedMessages: []string{
				"[log] The following are log messages\n",
				"[log] This one is\n\n on multiple\n\n lines",
				"[log] In total there should be 3 events\n",
			},
		},
		"humanize max_bytes, multiline XML": {
			lines: `<Event><Data>
	A
	B
	C</Data></Event>
<Event><Data>
	D
	E
	F</Data></Event>
`,
			parsers: map[string]interface{}{
				"max_bytes":       "4 KiB",
				"line_terminator": "auto",
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"multiline": map[string]interface{}{
							"match":   "after",
							"negate":  true,
							"pattern": "^<Event",
						},
					},
				},
			},
			expectedMessages: []string{
				"<Event><Data>\n\n\tA\n\n\tB\n\n\tC</Data></Event>\n",
				"<Event><Data>\n\n\tD\n\n\tE\n\n\tF</Data></Event>\n",
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(test.parsers)
			var c inputParsersConfig
			err := cfg.Unpack(&c)
			require.NoError(t, err)

			p := c.Parsers.Create(testReader(test.lines))

			i := 0
			msg, err := p.Next()
			for err == nil {
				require.Equal(t, test.expectedMessages[i], string(msg.Content))
				i++
				msg, err = p.Next()
			}
		})
	}
}
