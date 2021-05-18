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

package filestream

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/multiline"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

func TestParsersConfigAndReading(t *testing.T) {
	tests := map[string]struct {
		lines            string
		parsers          map[string]interface{}
		expectedMessages []string
		expectedError    string
	}{
		"no parser, no error": {
			lines: "line 1\nline 2\n",
			parsers: map[string]interface{}{
				"paths": []string{"dummy_path"},
			},
			expectedMessages: []string{"line 1\n", "line 2\n"},
		},
		"correct multiline parser": {
			lines: "line 1.1\nline 1.2\nline 1.3\nline 2.1\nline 2.2\nline 2.3\n",
			parsers: map[string]interface{}{
				"paths": []string{"dummy_path"},
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"multiline": map[string]interface{}{
							"type":        "count",
							"count_lines": 3,
						},
					},
				},
			},
			expectedMessages: []string{
				"line 1.1\n\nline 1.2\n\nline 1.3\n",
				"line 2.1\n\nline 2.2\n\nline 2.3\n",
			},
		},
		"multiline docker logs parser": {
			lines: `{"log":"[log] The following are log messages\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"[log] This one is\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":" on multiple\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":" lines","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
{"log":"[log] In total there should be 3 events\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}
`,
			parsers: map[string]interface{}{
				"paths": []string{"dummy_path"},
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
		"non existent parser configuration": {
			parsers: map[string]interface{}{
				"paths": []string{"dummy_path"},
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"no_such_parser": nil,
					},
				},
			},
			expectedError: ErrNoSuchParser.Error(),
		},
		"invalid multiline parser configuration is caught before parser creation": {
			parsers: map[string]interface{}{
				"paths": []string{"dummy_path"},
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"multiline": map[string]interface{}{
							"match": "after",
						},
					},
				},
			},
			expectedError: multiline.ErrMissingPattern.Error(),
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := defaultConfig()
			parsersConfig := common.MustNewConfigFrom(test.parsers)
			err := parsersConfig.Unpack(&cfg)
			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), test.expectedError)
				return
			}

			p, err := newParsers(testReader(test.lines), parserConfig{lineTerminator: readfile.AutoLineTerminator, maxBytes: 64}, cfg.Reader.Parsers)

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

func TestJSONParsersWithFields(t *testing.T) {
	tests := map[string]struct {
		message         reader.Message
		config          map[string]interface{}
		expectedMessage reader.Message
	}{
		"no postprocesser, no processing": {
			message: reader.Message{
				Content: []byte("line 1"),
			},
			config: map[string]interface{}{
				"paths": []string{"dummy_path"},
			},
			expectedMessage: reader.Message{
				Content: []byte("line 1"),
			},
		},
		"JSON post processer with keys_under_root": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\"}"),
				Fields:  common.MapStr{},
			},
			config: map[string]interface{}{
				"paths": []string{"dummy_path"},
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"target": "",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: common.MapStr{
					"key": "value",
				},
			},
		},
		"JSON post processer with document ID": {
			message: reader.Message{
				Content: []byte("{\"key\":\"value\", \"my-id-field\":\"my-id\"}"),
				Fields:  common.MapStr{},
			},
			config: map[string]interface{}{
				"paths": []string{"dummy_path"},
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"target":      "",
							"document_id": "my-id-field",
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: common.MapStr{
					"key": "value",
				},
				Meta: common.MapStr{
					"_id": "my-id",
				},
			},
		},
		"JSON post processer with overwrite keys and under root": {
			message: reader.Message{
				Content: []byte("{\"key\": \"value\"}"),
				Fields: common.MapStr{
					"key":       "another-value",
					"other-key": "other-value",
				},
			},
			config: map[string]interface{}{
				"paths": []string{"dummy_path"},
				"parsers": []map[string]interface{}{
					map[string]interface{}{
						"ndjson": map[string]interface{}{
							"target":         "",
							"overwrite_keys": true,
						},
					},
				},
			},
			expectedMessage: reader.Message{
				Content: []byte(""),
				Fields: common.MapStr{
					"key":       "value",
					"other-key": "other-value",
				},
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := defaultConfig()
			common.MustNewConfigFrom(test.config).Unpack(&cfg)
			p, err := newParsers(msgReader(test.message), parserConfig{lineTerminator: readfile.AutoLineTerminator, maxBytes: 64}, cfg.Reader.Parsers)
			if err != nil {
				t.Fatalf("failed to init parser: %+v", err)
			}

			msg, _ := p.Next()
			require.Equal(t, test.expectedMessage, msg)
		})
	}

}

func testReader(lines string) reader.Reader {
	encF, _ := encoding.FindEncoding("")
	reader := strings.NewReader(lines)
	enc, err := encF(reader)
	if err != nil {
		panic(err)
	}
	r, err := readfile.NewEncodeReader(ioutil.NopCloser(reader), readfile.Config{
		Codec:      enc,
		BufferSize: 1024,
		Terminator: readfile.AutoLineTerminator,
		MaxBytes:   1024,
	})
	if err != nil {
		panic(err)
	}

	return r
}

func msgReader(m reader.Message) reader.Reader {
	return &messageReader{
		message: m,
	}
}

type messageReader struct {
	message reader.Message
	read    bool
}

func (r *messageReader) Next() (reader.Message, error) {
	if r.read {
		return reader.Message{}, io.EOF
	}
	r.read = true
	return r.message, nil
}

func (r *messageReader) Close() error {
	r.message = reader.Message{}
	r.read = false
	return nil
}
