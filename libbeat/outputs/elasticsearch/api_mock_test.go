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

package elasticsearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

func ElasticsearchMock(code int, body []byte) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" { // send ok and a minimal JSON on ping
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"version":{"number":"5.0.0"}}`))
			return
		}

		w.WriteHeader(code)
		if body != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
		}
	}))

	return server
}

func TestOneHostSuccessResp(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
	expectedResp, _ := json.Marshal(QueryResult{Ok: true, Index: index, Type: "test", ID: "1", Version: 1, Created: true})

	server := ElasticsearchMock(200, expectedResp)

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, resp, err := client.Index(index, "test", "1", params, body)
	if err != nil {
		t.Errorf("Index() returns error: %s", err)
	}
	if !resp.Created {
		t.Errorf("Index() fails: %s", resp)
	}
}

func TestOneHost500Resp(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}

	server := ElasticsearchMock(http.StatusInternalServerError, []byte("Something wrong happened"))

	client := newTestClient(server.URL)
	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err = client.Index(index, "test", "1", params, body)

	if err == nil {
		t.Errorf("Index() should return error.")
	}

	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Errorf("Should return <500 Internal Server Error> instead of %v", err)
	}
}

func TestOneHost503Resp(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}

	server := ElasticsearchMock(503, []byte("Something wrong happened"))

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err := client.Index(index, "test", "1", params, body)
	if err == nil {
		t.Errorf("Index() should return error.")
	}

	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("Should return <503 Service Unavailable> instead of %v", err)
	}
}
