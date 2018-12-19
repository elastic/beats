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

// +build !integration

package encoding

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestUtf16BOMEncodings(t *testing.T) {
	expectedLE := utf16Map[littleEndian]
	expectedBE := utf16Map[bigEndian]

	var tests = []struct {
		name             string
		testEndianness   unicode.Endianness
		testBOMPolicy    unicode.BOMPolicy
		expectedEncoding Encoding
		expectedError    error
		expectedOffset   int
	}{
		{"utf-16-bom",
			unicode.BigEndian, unicode.ExpectBOM, expectedBE, nil, 2},
		{"utf-16-bom",
			unicode.BigEndian, unicode.IgnoreBOM, nil, unicode.ErrMissingBOM, 0},
		{"utf-16-bom",
			unicode.LittleEndian, unicode.ExpectBOM, expectedLE, nil, 2},
		{"utf-16-bom",
			unicode.LittleEndian, unicode.IgnoreBOM, nil, unicode.ErrMissingBOM, 0},

		// big endian based encoding
		{"utf-16be-bom",
			unicode.BigEndian, unicode.ExpectBOM, expectedBE, nil, 2},
		{"utf-16be-bom",
			unicode.BigEndian, unicode.IgnoreBOM, expectedBE, nil, 0},
		{"utf-16be-bom",
			unicode.LittleEndian, unicode.ExpectBOM, expectedLE, nil, 2},

		// little endian baed encoding
		{"utf-16le-bom",
			unicode.LittleEndian, unicode.ExpectBOM, expectedLE, nil, 2},
		{"utf-16le-bom",
			unicode.LittleEndian, unicode.IgnoreBOM, expectedLE, nil, 0},
		{"utf-16le-bom",
			unicode.BigEndian, unicode.ExpectBOM, expectedBE, nil, 2},
	}

	text := []byte("hello world")

	for _, test := range tests {
		t.Logf("testing: codec=%v, bigendian=%v, bomPolicy=%v",
			test.name, test.testEndianness, test.testBOMPolicy)

		buf := bytes.NewBuffer(nil)
		writeEncoding := unicode.UTF16(test.testEndianness, test.testBOMPolicy)
		writer := transform.NewWriter(buf, writeEncoding.NewEncoder())
		writer.Write(text)
		writer.Close()

		rawReader := bytes.NewReader(buf.Bytes())
		contentLen := rawReader.Len()
		encodingFactory, ok := FindEncoding(test.name)
		if !ok {
			t.Errorf("Failed to load encoding: %v", test.name)
			continue
		}

		encoding, err := encodingFactory(rawReader)
		contentOffset := contentLen - rawReader.Len()

		assert.Equal(t, test.expectedEncoding, encoding)
		assert.Equal(t, test.expectedError, err)
		assert.Equal(t, test.expectedOffset, contentOffset)
		if err == nil {
			reader := transform.NewReader(rawReader, encoding.NewDecoder())
			content, _ := ioutil.ReadAll(reader)
			assert.Equal(t, text, content)
		}
	}
}
