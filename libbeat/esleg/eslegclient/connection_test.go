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
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common/productorigin"
	"github.com/stretchr/testify/require"
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

	req, err := http.NewRequest("GET", "http://fakehost/some/path", nil)
	require.NoError(t, err)

	_, _, err = conn.execHTTPRequest(req)
	require.NoError(t, err)

	require.Equal(t, "ApiKey "+encoded, httpClient.Req.Header.Get("Authorization"))
}

type mockClient struct {
	Req *http.Request
}

func (c *mockClient) Do(req *http.Request) (*http.Response, error) {
	c.Req = req

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

		req, err := http.NewRequest("GET", "http://fakehost/some/path", nil)
		require.NoError(t, err)
		_, _, err = conn.execHTTPRequest(req)
		require.NoError(t, err)

		require.Equal(t, req.Header, http.Header(td.expected))
	}
}
