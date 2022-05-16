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

//go:build !integration
// +build !integration

package eslegclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestOneHostSuccessResp_Bulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	expectedResp := []byte(`{"took":7,"errors":false,"items":[]}`)

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server := ElasticsearchMock(200, expectedResp)

	client := newTestConnection(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err := client.Bulk(context.Background(), index, "type1", params, body)
	if err != nil {
		t.Errorf("Bulk() returns error: %s", err)
	}
}

func TestOneHost500Resp_Bulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server := ElasticsearchMock(http.StatusInternalServerError, []byte("Something wrong happened"))

	client := newTestConnection(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err := client.Bulk(context.Background(), index, "type1", params, body)
	if err == nil {
		t.Errorf("Bulk() should return error.")
	}

	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Errorf("Should return <500 Internal Server Error> instead of %v", err)
	}
}

func TestOneHost503Resp_Bulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server := ElasticsearchMock(503, []byte("Something wrong happened"))

	client := newTestConnection(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err := client.Bulk(context.Background(), index, "type1", params, body)
	if err == nil {
		t.Errorf("Bulk() should return error.")
	}

	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("Should return <503 Service Unavailable> instead of %v", err)
	}
}

func TestEnforceParameters(t *testing.T) {
	// Prepare the test bulk request.
	index := "what"

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	tests := map[string]struct {
		preconfigured map[string]string
		reqParams     map[string]string
		expected      map[string]string
	}{
		"Preconfigured parameters are applied to bulk requests": {
			preconfigured: map[string]string{
				"hello": "world",
			},
			expected: map[string]string{
				"hello": "world",
			},
		},
		"Preconfigured and local parameters are merged": {
			preconfigured: map[string]string{
				"hello": "world",
			},
			reqParams: map[string]string{
				"foo": "bar",
			},
			expected: map[string]string{
				"hello": "world",
				"foo":   "bar",
			},
		},
		"Local parameters only": {
			reqParams: map[string]string{
				"foo": "bar",
			},
			expected: map[string]string{
				"foo": "bar",
			},
		},
		"no parameters": {
			expected: map[string]string{},
		},
		"Local overrides preconfigured parameters": {
			preconfigured: map[string]string{
				"foo": "world",
			},
			reqParams: map[string]string{
				"foo": "bar",
			},
			expected: map[string]string{
				"foo": "bar",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client, _ := NewConnection(ConnectionSettings{
				Parameters: test.preconfigured,
				URL:        "http://localhost",
			})

			client.Encoder = NewJSONEncoder(nil, false)

			var recParams url.Values
			errShort := errors.New("shortcut")

			client.HTTP = &reqInspector{
				assert: func(req *http.Request) (*http.Response, error) {
					recParams = req.URL.Query()
					return nil, errShort
				},
			}

			_, _, err := client.Bulk(context.Background(), index, "type1", test.reqParams, body)
			require.Equal(t, errShort, err)
			require.Equal(t, len(recParams), len(test.expected))

			for k, v := range test.expected {
				assert.Equal(t, recParams.Get(k), v)
			}
		})
	}
}

type reqInspector struct {
	assert func(req *http.Request) (*http.Response, error)
}

func (r *reqInspector) Do(req *http.Request) (*http.Response, error) {
	return r.assert(req)
}

func (r *reqInspector) CloseIdleConnections() {
}
