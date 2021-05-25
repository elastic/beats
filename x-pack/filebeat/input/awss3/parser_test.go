// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

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
			lines:            "line 1\nline 2\n",
			expectedMessages: []string{"line 1\n", "line 2\n"},
		},
		"correct multiline parser": {
			lines: "line 1.1\nline 1.2\nline 1.3\nline 2.1\nline 2.2\nline 2.3\n",
			parsers: map[string]interface{}{
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
			cfg.QueueURL = "https://example.com"
			parsersConfig := common.MustNewConfigFrom(test.parsers)
			err := parsersConfig.Unpack(&cfg)
			if test.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedError)
				return
			}

			p, err := newParsers(testReader(test.lines), parserConfig{lineTerminator: readfile.AutoLineTerminator, maxBytes: 64}, cfg.ReaderConfig.Parsers)

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
