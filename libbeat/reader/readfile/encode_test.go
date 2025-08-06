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
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestEncodeLines(t *testing.T) {
	testCases := map[string]struct {
		Input  []byte
		Output []string
	}{
		"simple":                  {[]byte("testing simple line\n"), []string{"testing simple line\n"}},
		"multiline":               {[]byte("testing\nmultiline\n"), []string{"testing\n", "multiline\n"}},
		"bom-on-first-bytes":      {[]byte("\xef\xbb\xbftesting simple line\n"), []string{"testing simple line\n"}},
		"bom-on-each-bytes":       {[]byte("\xef\xbb\xbftesting\n\xef\xbb\xbfmultiline\n"), []string{"testing\n", "multiline\n"}},
		"bom-in-the-middle-bytes": {[]byte("testing simple \xef\xbb\xbfline\n"), []string{"testing simple \xef\xbb\xbfline\n"}},
		"not-a-bom-bytes":         {[]byte("\xef\xbf\xbbtesting simple line\n"), []string{"\xef\xbf\xbbtesting simple line\n"}},
		"bom-on-first-rune":       {[]byte("\uFEFFtesting simple line\n"), []string{"testing simple line\n"}},
		"bom-on-each-rune":        {[]byte("\uFEFFtesting\n\uFEFFmultiline\n"), []string{"testing\n", "multiline\n"}},
		"bom-in-the-middle-rune":  {[]byte("testing simple \uFEFFline\n"), []string{"testing simple \uFEFFline\n"}},
		"not-a-bom-rune":          {[]byte("\uFFEFtesting simple line\n"), []string{"\uFFEFtesting simple line\n"}},
		// This final test is included for completeness. It is never possible for
		// a line obtained in the Next method to have a BOM suffix since all
		// lines are new-line terminated, thus the new-line blocks the BOM
		// from being identified as a suffix even in the case that is was
		// actively looked for.
		"not-a-prefix-bytes": {[]byte("testing simple line\xef\xbf\xbb\n"), []string{"testing simple line\xef\xbf\xbb\n"}},
		"not-a-prefix-rune":  {[]byte("testing simple line\uFEFF\n"), []string{"testing simple line\uFEFF\n"}},
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
				Terminator: LineFeed,
			}
			er, err := NewEncodeReader(r, config, logptest.NewTestingLogger(t, ""))
			assert.Nil(t, err, "failed to create new encoder: %v", err)

			var output []string
			for {
				msg, err := er.Next()
				if err != nil {
					break
				}
				output = append(output, string(msg.Content))
			}

			assert.Equal(t, testCase.Output, output)
		})
	}
}
