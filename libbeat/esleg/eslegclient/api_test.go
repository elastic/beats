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

// Need for unit and integration tests
package eslegclient

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/logp"
)

func GetValidQueryResult() QueryResult {
	result := QueryResult{
		Ok:    true,
		Index: "testIndex",
		Type:  "testType",
		ID:    "12",
		Source: []byte(`{
			"ok": true,
			"_type":"testType",
			"_index":"testIndex",
			"_id":"12",
			"_version": 2,
			"found": true,
			"exists": true,
			"created": true,
			"lastname":"ruflin",
			"firstname": "nicolas"}`,
		),
		Version: 2,
		Found:   true,
		Exists:  true,
		Created: true,
		Matches: []string{"abc", "def"},
	}

	return result
}

func GetValidSearchResults() SearchResults {
	hits := Hits{
		Total: Total{Value: 10, Relation: "eq"},
		Hits:  nil,
	}

	results := SearchResults{
		Took: 19,
		Shards: []byte(`{
    		"total" : 3,
    		"successful" : 2,
    		"failed" : 1
  		}`),
		Hits: hits,
		Aggs: nil,
	}

	return results
}

func TestReadQueryResult(t *testing.T) {
	queryResult := GetValidQueryResult()

	json := queryResult.Source
	result, err := readQueryResult(json)

	assert.NoError(t, err)
	assert.Equal(t, queryResult.Ok, result.Ok)
	assert.Equal(t, queryResult.Index, result.Index)
	assert.Equal(t, queryResult.Type, result.Type)
	assert.Equal(t, queryResult.ID, result.ID)
	assert.Equal(t, queryResult.Version, result.Version)
	assert.Equal(t, queryResult.Found, result.Found)
	assert.Equal(t, queryResult.Exists, result.Exists)
	assert.Equal(t, queryResult.Created, result.Created)
}

// Check empty query result object
func TestReadQueryResult_empty(t *testing.T) {
	result, err := readQueryResult(nil)
	assert.Nil(t, result)
	assert.NoError(t, err)
}

// Check invalid query result object
func TestReadQueryResult_invalid(t *testing.T) {
	// Invalid json string
	json := []byte(`{"name":"ruflin","234"}`)

	result, err := readQueryResult(json)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestReadSearchResult(t *testing.T) {
	t.Run("search results response from 7.0", func(t *testing.T) {
		resultsObject := GetValidSearchResults()
		json := []byte(`{
  		"took" : 19,
  		"_shards" : {
    		"total" : 3,
    		"successful" : 2,
    		"failed" : 1
  		},
			"hits" : { "total": { "value": 10, "relation": "eq" } },
  		"aggs" : {}
  	}`)

		results, err := readSearchResult(json)

		assert.NoError(t, err)
		assert.Equal(t, resultsObject.Took, results.Took)
		assert.Equal(t, resultsObject.Hits, results.Hits)
		assert.Equal(t, resultsObject.Shards, results.Shards)
		assert.Equal(t, resultsObject.Aggs, results.Aggs)
	})

	t.Run("search results response from 6.0", func(t *testing.T) {
		resultsObject := GetValidSearchResults()
		json := []byte(`{
  		"took" : 19,
  		"_shards" : {
    		"total" : 3,
    		"successful" : 2,
    		"failed" : 1
  		},
			"hits" : { "total": 10 },
  		"aggs" : {}
  	}`)

		results, err := readSearchResult(json)

		assert.NoError(t, err)
		assert.Equal(t, resultsObject.Took, results.Took)
		assert.Equal(t, resultsObject.Hits, results.Hits)
		assert.Equal(t, resultsObject.Shards, results.Shards)
		assert.Equal(t, resultsObject.Aggs, results.Aggs)
	})
}

func TestReadSearchResult_empty(t *testing.T) {
	results, err := readSearchResult(nil)
	assert.Nil(t, results)
	assert.NoError(t, err)
}

func TestReadSearchResult_invalid(t *testing.T) {
	// Invalid json string
	json := []byte(`{"took":"19","234"}`)

	results, err := readSearchResult(json)
	assert.Nil(t, results)
	assert.Error(t, err)
}

func newTestConnection(url string) *Connection {
	conn, _ := NewConnection(ConnectionSettings{
		URL: url,
	})
	conn.Encoder = NewJSONEncoder(nil, false)
	return conn
}

func (r QueryResult) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		logp.L().Warnf("failed to marshal QueryResult (%+v): %#v", err, r)
		return "ERROR"
	}
	return string(out)
}
