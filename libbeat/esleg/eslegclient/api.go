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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/elastic/elastic-agent-libs/version"
)

// QueryResult contains the result of a query.
type QueryResult struct {
	Ok           bool            `json:"ok"`
	Index        string          `json:"_index"`
	Type         string          `json:"_type"`
	ID           string          `json:"_id"`
	Source       json.RawMessage `json:"_source"`
	Version      int             `json:"_version"`
	Exists       bool            `json:"exists"`
	Found        bool            `json:"found"`   // Only used prior to ES 6. You must also check for Result == "found".
	Created      bool            `json:"created"` // Only used prior to ES 6. You must also check for Result == "created".
	Result       string          `json:"result"`  // Only used in ES 6+.
	Acknowledged bool            `json:"acknowledged"`
	Matches      []string        `json:"matches"`
}

// SearchResults contains the results of a search.
type SearchResults struct {
	Took   int                        `json:"took"`
	Shards json.RawMessage            `json:"_shards"`
	Hits   Hits                       `json:"hits"`
	Aggs   map[string]json.RawMessage `json:"aggregations"`
}

// Hits contains the hits.
type Hits struct {
	Total Total
	Hits  []json.RawMessage `json:"hits"`
}

// Total contains the number of element fetched and the relation.
type Total struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

// UnmarshalJSON correctly unmarshal the hits response between ES 6.0 and ES 7.0.
func (t *Total) UnmarshalJSON(b []byte) error {
	value := struct {
		Value    int    `json:"value"`
		Relation string `json:"relation"`
	}{}

	if err := json.Unmarshal(b, &value); err == nil {
		*t = value
		return nil
	}

	// fallback for Elasticsearch < 7
	if i, err := strconv.Atoi(string(b)); err == nil {
		*t = Total{Value: i, Relation: "eq"}
		return nil
	}

	return fmt.Errorf("could not unmarshal JSON value '%s'", string(b))
}

// CountResults contains the count of results.
type CountResults struct {
	Count  int             `json:"count"`
	Shards json.RawMessage `json:"_shards"`
}

func withQueryResult(status int, resp []byte, err error) (int, *QueryResult, error) {
	if err != nil {
		return status, nil, fmt.Errorf("Elasticsearch response: %s: %w", resp, err)
	}
	result, err := readQueryResult(resp)
	return status, result, err
}

func readQueryResult(obj []byte) (*QueryResult, error) {
	var result QueryResult
	if obj == nil {
		return nil, nil
	}

	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func readSearchResult(obj []byte) (*SearchResults, error) {
	var result SearchResults
	if obj == nil {
		return nil, nil
	}

	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func readCountResult(obj []byte) (*CountResults, error) {
	if obj == nil {
		return nil, nil
	}

	var result CountResults
	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// Index adds or updates a typed JSON document in a specified index, making it
// searchable. In case id is empty, a new id is created over a HTTP POST request.
// Otherwise, a HTTP PUT request is issued.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-index_.html
func (conn *Connection) Index(
	index, docType, id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}
	return withQueryResult(conn.apiCall(method, index, docType, id, "", params, body))
}

// Ingest pushes a pipeline of updates.
func (conn *Connection) Ingest(
	index, docType, pipeline, id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}
	return withQueryResult(conn.apiCall(method, index, docType, id, pipeline, params, body))
}

// Refresh an index. Call this after doing inserts or creating/deleting
// indexes in unit tests.
func (conn *Connection) Refresh(index string) (int, *QueryResult, error) {
	return withQueryResult(conn.apiCall("POST", index, "", "_refresh", "", nil, nil))
}

// CreateIndex creates a new index, optionally with settings and mappings passed in
// the body.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html
func (conn *Connection) CreateIndex(index string, body interface{}) (int, *QueryResult, error) {
	return withQueryResult(conn.apiCall("PUT", index, "", "", "", nil, body))
}

// IndexExists checks if an index exists.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-exists.html
func (conn *Connection) IndexExists(index string) (int, error) {
	status, _, err := conn.apiCall("HEAD", index, "", "", "", nil, nil)
	return status, err
}

// Delete deletes a typed JSON document from a specific index based on its id.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-delete.html
func (conn *Connection) Delete(index string, docType string, id string, params map[string]string) (int, *QueryResult, error) {
	return withQueryResult(conn.apiCall("DELETE", index, docType, id, "", params, nil))
}

// PipelineExists checks if a pipeline with name id already exists.
// Using: https://www.elastic.co/guide/en/elasticsearch/reference/current/get-pipeline-api.html
func (conn *Connection) PipelineExists(id string) (bool, error) {
	status, _, err := conn.apiCall("GET", "_ingest", "pipeline", id, "", nil, nil)
	if status == 404 {
		return false, nil
	}
	return status == 200, err
}

// CreatePipeline create a new ingest pipeline with name id.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/put-pipeline-api.html
func (conn *Connection) CreatePipeline(
	id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	return withQueryResult(conn.apiCall("PUT", "_ingest", "pipeline", id, "", params, body))
}

// DeletePipeline deletes an ingest pipeline by id.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-pipeline-api.html
func (conn *Connection) DeletePipeline(
	id string,
	params map[string]string,
) (int, *QueryResult, error) {
	return withQueryResult(conn.apiCall("DELETE", "_ingest", "pipeline", id, "", params, nil))
}

// SearchURI executes a search request using a URI by providing request parameters.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/search-uri-request.html
func (conn *Connection) SearchURI(index string, docType string, params map[string]string) (int, *SearchResults, error) {
	return conn.SearchURIWithBody(index, docType, params, nil)
}

// SearchURIWithBody executes a search request using a URI by providing request
// parameters and a request body.
func (conn *Connection) SearchURIWithBody(
	index string,
	docType string,
	params map[string]string,
	body interface{},
) (int, *SearchResults, error) {
	if !conn.version.LessThan(&version.V{Major: 8}) {
		docType = ""
	}
	status, resp, err := conn.apiCall("GET", index, docType, "_search", "", params, body)
	if err != nil {
		return status, nil, err
	}
	result, err := readSearchResult(resp)
	return status, result, err
}

// CountSearchURI counts the results for a search request.
func (conn *Connection) CountSearchURI(
	index string, docType string,
	params map[string]string,
) (int, *CountResults, error) {
	status, resp, err := conn.apiCall("GET", index, docType, "_count", "", params, nil)
	if err != nil {
		return status, nil, err
	}
	result, err := readCountResult(resp)
	return status, result, err
}

func (conn *Connection) apiCall(
	method, index, docType, id, pipeline string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {
	path, err := makePath(index, docType, id)
	if err != nil {
		return 0, nil, err
	}
	return conn.Request(method, path, pipeline, params, body)
}
