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

package eslegclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/productorigin"
	"github.com/elastic/beats/v7/libbeat/version"
)

func TestAPIKeyEncoding(t *testing.T) {
	apiKey := "foobar"
	encoded := base64.StdEncoding.EncodeToString([]byte(apiKey))

	conn, err := NewConnection(ConnectionSettings{
		APIKey: apiKey,
	})
	require.NoError(t, err)

	httpClient := newMockClient()
	conn.HTTP = httpClient

	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://fakehost/some/path", nil)
	require.NoError(t, err)

	_, _, err = conn.execHTTPRequest(req)
	require.NoError(t, err)

	require.Equal(t, "ApiKey "+encoded, httpClient.Req.Header.Get("Authorization"))
}

type mockClient struct {
	Req *http.Request
	Res *http.Response
}

func (c *mockClient) Do(req *http.Request) (*http.Response, error) {
	c.Req = req

	if c.Res != nil {
		return c.Res, nil
	}

	r := bytes.NewReader([]byte("HTTP/1.1 200 OK\n\nHello, world"))
	return http.ReadResponse(bufio.NewReader(r), req)
}

func (c *mockClient) CloseIdleConnections() {}

func newMockClient() *mockClient {
	return &mockClient{}
}

func TestHeaders(t *testing.T) {
	for _, td := range []struct {
		input    map[string]string
		expected map[string][]string
	}{
		{input: map[string]string{
			"Accept":             "application/vnd.elasticsearch+json;compatible-with=7",
			"Content-Type":       "application/vnd.elasticsearch+json;compatible-with=7",
			productorigin.Header: "elastic-product",
			"X-My-Header":        "true"},
			expected: map[string][]string{
				"Accept":             {"application/vnd.elasticsearch+json;compatible-with=7"},
				"Content-Type":       {"application/vnd.elasticsearch+json;compatible-with=7"},
				productorigin.Header: {"elastic-product"},
				"X-My-Header":        {"true"}}},
		{input: map[string]string{
			"X-My-Header": "true"},
			expected: map[string][]string{
				"Accept":             {"application/json"},
				productorigin.Header: {productorigin.Beats},
				"X-My-Header":        {"true"}}},
	} {
		conn, err := NewConnection(ConnectionSettings{
			Headers: td.input,
		})
		require.NoError(t, err)

		httpClient := newMockClient()
		conn.HTTP = httpClient

		req, err := http.NewRequestWithContext(context.Background(), "GET", "http://fakehost/some/path", nil)
		require.NoError(t, err)
		_, _, err = conn.execHTTPRequest(req)
		require.NoError(t, err)

		require.Equal(t, req.Header, http.Header(td.expected))

	}
}

func TestUserAgentHeader(t *testing.T) {

	// remove some randomness from this test
	version.SetPackageVersion("8.15")

	cases := []struct {
		connSettings ConnectionSettings
		expectedUA   string
		name         string
	}{
		{
			name: "test-ua-set",
			connSettings: ConnectionSettings{
				Beatname:  "testbeat",
				UserAgent: "Agent/8.15",
			},
			expectedUA: "Agent/8.15",
		},
		{
			name: "beatname-fallback",
			connSettings: ConnectionSettings{
				Beatname: "testbeat",
			},
			expectedUA: "testbeat/8.15",
		},
		{
			name:         "libbeat-fallback",
			connSettings: ConnectionSettings{},
			expectedUA:   "Libbeat/8.15",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.UserAgent(), testCase.expectedUA) {
					t.Errorf("User-Agent must be '%s', got '%s'", testCase.expectedUA, r.UserAgent())
				}
				_, _ = w.Write([]byte("{}"))
			}))
			defer server.Close()
			testCase.connSettings.URL = server.URL
			conn, err := NewConnection(testCase.connSettings)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			require.NoError(t, conn.Connect(ctx), "conn.Connect must not return an error")
		})
	}
}

func BenchmarkExecHTTPRequest(b *testing.B) {
	sizes := []int{
		100,             // 100 bytes
		10 * 1024,       // 10KB
		100 * 1024,      // 100KB
		1 * 1024 * 1024, // 1MB
	}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size %d", size), func(b *testing.B) {
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

			for _, tc := range cases {
				b.Run(tc.name, func(b *testing.B) {
					conn, err := NewConnection(ConnectionSettings{
						Headers: map[string]string{
							"Accept":       "application/vnd.elasticsearch+json;compatible-with=7",
							"Content-Type": "application/vnd.elasticsearch+json;compatible-with=7",
						},
					})
					require.NoError(b, err)

					httpClient := newMockClient()
					httpClient.Res = tc.resp
					conn.HTTP = httpClient

					var bb []byte
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err = content.Seek(0, io.SeekStart)
						require.NoError(b, err)
						req, err := http.NewRequestWithContext(context.Background(), "GET", "http://fakehost/some/path", nil)
						require.NoError(b, err)
						_, bb, err = conn.execHTTPRequest(req)
						require.NoError(b, err)
						require.Equal(b, generated, bb)
					}
				})
			}
		})
	}
}
