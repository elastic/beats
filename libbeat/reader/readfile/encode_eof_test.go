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

package readfile

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

func TestEncodeEof(t *testing.T) {
	testCases := map[string]struct {
		Input  []byte
		Output []string
	}{
		"simple":            {[]byte("testing simple line"), []string{"testing simple line"}},
		"multiline":         {[]byte("testing\nmultiline"), []string{"testing\nmultiline"}},
		"bom-on-first":      {[]byte("\xef\xbb\xbftesting simple line"), []string{"testing simple line"}},
		"bom-on-each":       {[]byte("\xef\xbb\xbftesting\n\xef\xbb\xbfmultiline"), []string{"testing\nmultiline"}},
		"bom-in-the-middle": {[]byte("testing simple \xef\xbb\xbfline"), []string{"testing simple line"}},
	}

	bufferSize := 1000
	encFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		t.Fatal("failed to initiate encoding")
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			r := ioutil.NopCloser(bytes.NewReader(testCase.Input))
			codec, err := encFactory(r)
			assert.Nil(t, err, "failed to initialize encoding: %v", err)

			config := Config{
				Codec:      codec,
				BufferSize: bufferSize,
				MaxBytes:   1000,
			}
			er := NewEncodeReaderEof(r, config)
			assert.Nil(t, err, "failed to create new encoder: %v", err)

			var output []string
			msg, err := er.Next()
			assert.Equal(t, io.EOF, err)
			output = append(output, string(msg.Content))

			assert.Equal(t, testCase.Output, output)
		})
	}
}
