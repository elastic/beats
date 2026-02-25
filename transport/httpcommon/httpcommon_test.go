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
	"net/http/httptest"
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
		"includes auth": {
			input: `
auth:
  api_key: test-key
timeout: 5s
`,
			expected: HTTPTransportSettings{
				Auth: &HTTPAuthorization{
					APIKey: "test-key",
				},
				Timeout: 5 * time.Second,
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

func TestReadAllWithLimit(t *testing.T) {
	size := 100
	body := bytes.Repeat([]byte{'a'}, size)
	cases := []struct {
		name    string
		resp    *http.Response
		limit   int64
		expBody []byte
		expErr  error
	}{
		{
			name: "reads known size without limit",
			resp: &http.Response{
				ContentLength: int64(size),
				Body:          io.NopCloser(bytes.NewBuffer(body)),
			},
			limit:   -1,
			expBody: body,
		},
		{
			name: "does not read known size if exceeds limit",
			resp: &http.Response{
				ContentLength: int64(size),
				Body:          io.NopCloser(bytes.NewBuffer(body)),
			},
			limit:  10,
			expErr: ErrResponseLimit,
		},
		{
			name: "reads unknown size without limit",
			resp: &http.Response{
				ContentLength: -1,
				Body:          io.NopCloser(bytes.NewBuffer(body)),
			},
			limit:   -1,
			expBody: body,
		},
		{
			name: "partially reads unknown size with limit",
			resp: &http.Response{
				ContentLength: -1,
				Body:          io.NopCloser(bytes.NewBuffer(body)),
			},
			limit:   10,
			expBody: body[:10],
		},
		{
			name: "supports empty with size=0",
			resp: &http.Response{
				ContentLength: 0,
			},
			limit:   -1,
			expBody: []byte{},
		},
		{
			name: "does not read the body if `No Content` status",
			resp: &http.Response{
				StatusCode: http.StatusNoContent,
			},
			limit:   -1,
			expBody: []byte{},
		},
		{
			name: "supports empty with unknown size",
			resp: &http.Response{
				ContentLength: -1,
				Body:          io.NopCloser(bytes.NewBuffer(nil)),
			},
			limit:   -1,
			expBody: []byte{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actBody, err := ReadAllWithLimit(tc.resp, tc.limit)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				require.Nil(t, actBody)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expBody, actBody)
		})
	}
}

func BenchmarkReadAll(b *testing.B) {
	sizes := []int{
		1024,        // 1KB
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

func Test_HTTPTransportSettings_RoundTripper(t *testing.T) {
	tests := []struct {
		name     string
		settings *HTTPTransportSettings
		handler  http.Handler
	}{{
		name: "with basic auth",
		settings: &HTTPTransportSettings{
			Auth: &HTTPAuthorization{
				Username: "test-user",
				Password: "test-password",
			},
		},
		handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			username, password, ok := req.BasicAuth()
			if !ok || username != "test-user" || password != "test-password" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}, {
		name: "with api key",
		settings: &HTTPTransportSettings{
			Auth: &HTTPAuthorization{
				APIKey: "test-key",
			},
		},
		handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "ApiKey test-key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}, {
		name: "with additional headers",
		settings: &HTTPTransportSettings{
			Auth: &HTTPAuthorization{
				Headers: []struct {
					Key   string `config:"key" yaml:"key,omitempty" json:"key,omitempty"`
					Value string `config:"value" yaml:"value,omitempty" json:"value,omitempty"`
				}{{
					Key:   "X-Authorization",
					Value: "test-extra",
				}, {
					Key:   "Other-Header",
					Value: "test-value",
				}},
			},
		},
		handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("X-Authorization") != "test-extra" || req.Header.Get("Other-Header") != "test-value" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt, err := tc.settings.RoundTripper()
			require.NoError(t, err)

			server := httptest.NewServer(tc.handler)
			defer server.Close()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func Test_HTTPAuthorization_ToMap(t *testing.T) {
	tests := []struct {
		name   string
		auth   *HTTPAuthorization
		expect map[string]string
	}{{
		name: "headers only",
		auth: &HTTPAuthorization{
			Headers: []struct {
				Key   string `config:"key" yaml:"key,omitempty" json:"key,omitempty"`
				Value string `config:"value" yaml:"value,omitempty" json:"value,omitempty"`
			}{{
				Key:   "header1",
				Value: "val1",
			}, {
				Key:   "header2",
				Value: "val2",
			}},
		},
		expect: map[string]string{
			"header1": "val1",
			"header2": "val2",
		},
	}, {
		name: "basic only",
		auth: &HTTPAuthorization{
			Username: "user",
			Password: "pass",
		},
		expect: map[string]string{
			"Authorization": "Basic dXNlcjpwYXNz",
		},
	}, {
		name: "basic with headers",
		auth: &HTTPAuthorization{
			Headers: []struct {
				Key   string `config:"key" yaml:"key,omitempty" json:"key,omitempty"`
				Value string `config:"value" yaml:"value,omitempty" json:"value,omitempty"`
			}{{
				Key:   "header1",
				Value: "val1",
			}, {
				Key:   "header2",
				Value: "val2",
			}},
			Username: "user",
			Password: "pass",
		},
		expect: map[string]string{
			"header1":       "val1",
			"header2":       "val2",
			"Authorization": "Basic dXNlcjpwYXNz",
		},
	}, {
		name: "api_key only",
		auth: &HTTPAuthorization{
			APIKey: "apiKeyVal",
		},
		expect: map[string]string{
			"Authorization": "ApiKey apiKeyVal",
		},
	}, {
		name: "api_key with headers",
		auth: &HTTPAuthorization{
			Headers: []struct {
				Key   string `config:"key" yaml:"key,omitempty" json:"key,omitempty"`
				Value string `config:"value" yaml:"value,omitempty" json:"value,omitempty"`
			}{{
				Key:   "header1",
				Value: "val1",
			}, {
				Key:   "header2",
				Value: "val2",
			}},
			APIKey: "apiKeyVal",
		},
		expect: map[string]string{
			"header1":       "val1",
			"header2":       "val2",
			"Authorization": "ApiKey apiKeyVal",
		},
	}, {
		name: "api_key preffered over basic",
		auth: &HTTPAuthorization{
			APIKey:   "apiKeyVal",
			Username: "user",
			Password: "pass",
		},
		expect: map[string]string{
			"Authorization": "ApiKey apiKeyVal",
		},
	}, {
		name: "api_key replaces Authorization custom header",
		auth: &HTTPAuthorization{
			Headers: []struct {
				Key   string `config:"key" yaml:"key,omitempty" json:"key,omitempty"`
				Value string `config:"value" yaml:"value,omitempty" json:"value,omitempty"`
			}{{
				Key:   "Authorization",
				Value: "val1",
			}},
			APIKey: "apiKeyVal",
		},
		expect: map[string]string{
			"Authorization": "ApiKey apiKeyVal",
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := tc.auth.ToMap()
			require.EqualValues(t, tc.expect, results)
		})
	}
}
