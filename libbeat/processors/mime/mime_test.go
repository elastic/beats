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

package mime

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestMimeType(t *testing.T) {
	tests := []struct {
		name         string
		expectedType string
		body         string
	}{
		{
			name:         "html",
			expectedType: "text/html; charset=utf-8",
			body:         "<html>Test</html>",
		},
		{
			name:         "pe",
			expectedType: "application/vnd.microsoft.portable-executable",
			body:         convertToData(t, "4d5a90000300000004000000ffff"),
		},
		{
			name:         "elf",
			expectedType: "application/x-executable",
			body:         convertToData(t, "7f454c460101010000000000000000000300030001000000f0dc01003400000080a318000000000034002000080028001e001d0001"),
		},
		{
			name:         "macho",
			expectedType: "application/x-mach-binary",
			body:         convertToData(t, "cffaedfe0700000103000000020000001000000058050000850020000000000019000000480000005f5f504147455a45524f"),
		},
		{
			name:         "json",
			expectedType: "application/json",
			body:         "{}",
		},
		{
			name:         "xml",
			expectedType: "text/xml",
			body:         "<test></test>",
		},
		{
			name:         "text",
			expectedType: "text/plain; charset=utf-8",
			body:         "Hello world!",
		},
		{
			name:         "png",
			expectedType: "image/png",
			body:         convertToData(t, "89504e470d0a1a0a0000000d494844520000025800000258080200000031040f8b0000000467414d410000b18f0bfc610500"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evt := beat.Event{
				Fields: common.MapStr{
					"http.request.body.content": test.body,
				},
			}
			p, err := New(common.MustNewConfigFrom(map[string]interface{}{}))
			require.NoError(t, err)
			observed, err := p.Run(&evt)
			require.NoError(t, err)
			enriched, err := observed.Fields.GetValue("http.request.mime_type")
			require.NoError(t, err)
			require.Equal(t, test.expectedType, enriched)
		})
	}
}

func TestMimeTypeFromTo(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": "hello world!",
		},
	}
	p, err := New(common.MustNewConfigFrom(map[string]interface{}{
		"from": "foo.bar.baz",
		"to":   "bar.baz.zoiks",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	enriched, err := observed.Fields.GetValue("bar.baz.zoiks")
	require.NoError(t, err)
	require.Equal(t, "text/plain; charset=utf-8", enriched)
}

func TestMimeTypeTestNoMatch(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"http.request.body.content": string([]byte{0, 0}),
		},
	}
	p, err := New(common.MustNewConfigFrom(map[string]interface{}{}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	hasKey, _ := observed.Fields.HasKey("http.request.mime_type")
	require.False(t, hasKey)
}

func convertToData(t *testing.T, sample string) string {
	t.Helper()
	decoded, err := hex.DecodeString(sample)
	if err != nil {
		t.Fatal(err)
	}
	return string(decoded)
}
