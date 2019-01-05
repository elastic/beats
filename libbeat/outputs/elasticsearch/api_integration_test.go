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

// +build integration

package elasticsearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
)

func TestIndex(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("beats-test-index-%d", os.Getpid())

	client := getTestingElasticsearch(t)

	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
	params := map[string]string{
		"refresh": "true",
	}
	_, resp, err := client.Index(index, "test", "1", params, body)
	if err != nil {
		t.Fatalf("Index() returns error: %s", err)
	}
	if !resp.Created && resp.Result != "created" {
		t.Fatalf("Index() fails: %s", resp)
	}

	body = map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}
	_, result, err := client.SearchURIWithBody(index, "", nil, map[string]interface{}{})
	if err != nil {
		t.Errorf("SearchUriWithBody() returns an error: %s", err)
	}
	if result.Hits.Total.Value != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total.Value)
	}

	params = map[string]string{
		"q": "user:test",
	}
	_, result, err = client.SearchURI(index, "test", params)
	if err != nil {
		t.Errorf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total.Value != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total.Value)
	}

	_, resp, err = client.Delete(index, "test", "1", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}

func TestIngest(t *testing.T) {
	type obj map[string]interface{}

	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	index := fmt.Sprintf("beats-test-ingest-%d", os.Getpid())
	pipeline := fmt.Sprintf("beats-test-pipeline-%d", os.Getpid())

	pipelineBody := obj{
		"description": "Test pipeline",
		"processors": []obj{
			{
				"lowercase": obj{
					"field": "testfield",
				},
			},
		},
	}

	client := getTestingElasticsearch(t)
	if client.Connection.version.Major < 5 {
		t.Skip("Skipping tests as pipeline not available in <5.x releases")
	}

	status, _, err := client.DeletePipeline(pipeline, nil)
	if err != nil && status != http.StatusNotFound {
		t.Fatal(err)
	}

	exists, err := client.PipelineExists(pipeline)
	if err != nil {
		t.Fatal(err)
	}
	if exists == true {
		t.Fatalf("Test expected PipelineExists to return false for %v", pipeline)
	}

	_, resp, err := client.CreatePipeline(pipeline, nil, pipelineBody)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Acknowledged {
		t.Fatalf("Test pipeline %v not created", pipeline)
	}

	exists, err = client.PipelineExists(pipeline)
	if err != nil {
		t.Fatal(err)
	}
	if exists == false {
		t.Fatalf("Test expected PipelineExists to return true for %v", pipeline)
	}

	params := map[string]string{"refresh": "true"}
	_, resp, err = client.Ingest(index, "test", pipeline, "1", params, obj{
		"testfield": "TEST",
	})
	if err != nil {
		t.Fatalf("Ingest() returns error: %s", err)
	}
	if !resp.Created && resp.Result != "created" {
		t.Errorf("Ingest() fails: %s", resp)
	}

	// get _source field from indexed document
	_, docBody, err := client.apiCall("GET", index, "test", "1/_source", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	doc := struct {
		Field string `json:"testfield"`
	}{}
	err = json.Unmarshal(docBody, &doc)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "test", doc.Field)
}
