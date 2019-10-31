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
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
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

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, err := client.Bulk(index, "type1", params, body)
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

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, err := client.Bulk(index, "type1", params, body)
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

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, err := client.Bulk(index, "type1", params, body)
	if err == nil {
		t.Errorf("Bulk() should return error.")
	}

	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("Should return <503 Service Unavailable> instead of %v", err)
	}
}
