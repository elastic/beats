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

package httpcommon

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestUnpack(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected HTTPTransportSettings
	}{
		"blank": {
			input:    "",
			expected: HTTPTransportSettings{},
		},
		"idleConnectionTimeout": {
			input: `
idle_connection_timeout: 15s
`,
			expected: HTTPTransportSettings{IdleConnTimeout: 15 * time.Second},
		},
		"timeoutAndIdleConnectionTimeout": {
			input: `
idle_connection_timeout: 15s
timeout: 5s
`,
			expected: HTTPTransportSettings{
				IdleConnTimeout: 15 * time.Second,
				Timeout:         5 * time.Second,
			},
		},
		"ssl": {
			input: `
ssl:
  verification_mode: certificate
`,
			expected: HTTPTransportSettings{
				TLS: &tlscommon.Config{
					VerificationMode: tlscommon.VerifyCertificate,
				},
			},
		},
		"complex": {
			input: `
timeout: 5s
idle_connection_timeout: 15s
ssl:
  verification_mode: certificate
`,
			expected: HTTPTransportSettings{
				TLS: &tlscommon.Config{
					VerificationMode: tlscommon.VerifyCertificate,
				},
				IdleConnTimeout: 15 * time.Second,
				Timeout:         5 * time.Second,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := config.NewConfigFrom(tc.input)
			require.NoError(t, err)

			settings := HTTPTransportSettings{}
			err = cfg.Unpack(&settings)
			require.NoError(t, err)

			require.Equal(t, tc.expected, settings)
		})
	}
}

func TestReadAll(t *testing.T) {
	size := 100
	body := bytes.Repeat([]byte{'a'}, size)
	cases := []struct {
		name    string
		resp    *http.Response
		expBody []byte
	}{
		{
			name: "reads known size",
			resp: &http.Response{
				ContentLength: int64(size),
				Body:          io.NopCloser(bytes.NewBuffer(body)),
			},
			expBody: body,
		},
		{
			name: "reads unknown size",
			resp: &http.Response{
				ContentLength: -1,
				Body:          io.NopCloser(bytes.NewBuffer(body)),
			},
			expBody: body,
		},
		{
			name: "supports empty with size=0",
			resp: &http.Response{
				ContentLength: 0,
				Body:          io.NopCloser(bytes.NewBuffer(nil)),
			},
			expBody: []byte{},
		},
		{
			name: "supports empty with unknown size",
			resp: &http.Response{
				ContentLength: -1,
				Body:          io.NopCloser(bytes.NewBuffer(nil)),
			},
			expBody: []byte{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actBody, err := ReadAll(tc.resp)
			require.NoError(t, err)
			require.Equal(t, tc.expBody, actBody)
		})
	}
}

func BenchmarkReadAll(b *testing.B) {
	sizes := []int{
		100,         // 100 bytes
		100 * 1024,  // 100KB
		1024 * 1024, // 1MB
	}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size: %d", size), func(b *testing.B) {

			// emulate a file or an HTTP response
			generated := bytes.Repeat([]byte{'a'}, size)
			content := bytes.NewReader(generated)
			cases := []struct {
				name string
				resp *http.Response
			}{
				{
					name: "unknown length",
					resp: &http.Response{
						ContentLength: -1,
						Body:          io.NopCloser(content),
					},
				},
				{
					name: "known length",
					resp: &http.Response{
						ContentLength: int64(size),
						Body:          io.NopCloser(content),
					},
				},
			}

			b.ResetTimer()

			for _, tc := range cases {
				b.Run(tc.name, func(b *testing.B) {
					b.Run("io.ReadAll", func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							_, err := content.Seek(0, io.SeekStart) // reset
							require.NoError(b, err)
							data, err := io.ReadAll(tc.resp.Body)
							require.NoError(b, err)
							require.Equalf(b, size, len(data), "size does not match, expected %d, actual %d", size, len(data))
						}
					})
					b.Run("bytes.Buffer+io.Copy", func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							_, err := content.Seek(0, io.SeekStart) // reset
							require.NoError(b, err)
							data, err := ReadAll(tc.resp)
							require.NoError(b, err)
							require.Equalf(b, size, len(data), "size does not match, expected %d, actual %d", size, len(data))
						}
					})
				})
			}
		})
	}
}
