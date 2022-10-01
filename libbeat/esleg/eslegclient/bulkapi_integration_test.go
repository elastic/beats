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

//go:build integration
// +build integration

package eslegclient

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestBulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client := getTestingElasticsearch(t)
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
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

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err := client.Bulk(context.Background(), index, "", params, body)
	if err != nil {
		t.Fatalf("Bulk() returned error: %s", err)
	}

	params = map[string]string{
		"q": "field1:value1",
	}
	_, result, err := client.SearchURI(index, "type1", params)
	if err != nil {
		t.Fatalf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total.Value != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total.Value)
	}

	_, _, err = client.Delete(index, "", "", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}

func TestEmptyBulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client := getTestingElasticsearch(t)
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	body := make([]interface{}, 0, 10)

	params := map[string]string{
		"refresh": "true",
	}
	_, resp, err := client.Bulk(context.Background(), index, "", params, body)
	if err != nil {
		t.Fatalf("Bulk() returned error: %s", err)
	}
	if resp != nil {
		t.Errorf("Unexpected response: %s", resp)
	}
}

func TestBulkMoreOperations(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client := getTestingElasticsearch(t)
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},

		{
			"delete": map[string]interface{}{
				"_index": index,
				"_id":    "2",
			},
		},

		{
			"create": map[string]interface{}{
				"_index": index,
				"_id":    "3",
			},
		},
		{
			"field1": "value3",
		},

		{
			"update": map[string]interface{}{
				"_id":    "1",
				"_index": index,
			},
		},
		{
			"doc": map[string]interface{}{
				"field2": "value2",
			},
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	params := map[string]string{
		"refresh": "true",
	}
	_, resp, err := client.Bulk(context.Background(), index, "", params, body)
	if err != nil {
		t.Fatalf("Bulk() returned error: %s [%s]", err, resp)
	}

	params = map[string]string{
		"q": "field1:value3",
	}
	_, result, err := client.SearchURI(index, "type1", params)
	if err != nil {
		t.Fatalf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total.Value != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total.Value)
	}

	params = map[string]string{
		"q": "field2:value2",
	}
	_, result, err = client.SearchURI(index, "type1", params)
	if err != nil {
		t.Fatalf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total.Value != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total.Value)
	}

	_, _, err = client.Delete(index, "", "", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}
