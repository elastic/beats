package elasticsearch

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/logp"
)

type QueryResult struct {
	Ok           bool            `json:"ok"`
	Index        string          `json:"_index"`
	Type         string          `json:"_type"`
	ID           string          `json:"_id"`
	Source       json.RawMessage `json:"_source"`
	Version      int             `json:"_version"`
	Found        bool            `json:"found"`
	Exists       bool            `json:"exists"`
	Created      bool            `json:"created"`
	Acknowledged bool            `json:"acknowledged"`
	Matches      []string        `json:"matches"`
}

type SearchResults struct {
	Took   int                        `json:"took"`
	Shards json.RawMessage            `json:"_shards"`
	Hits   Hits                       `json:"hits"`
	Aggs   map[string]json.RawMessage `json:"aggregations"`
}

type Hits struct {
	Total int
	Hits  []json.RawMessage `json:"hits"`
}

type CountResults struct {
	Count  int             `json:"count"`
	Shards json.RawMessage `json:"_shards"`
}

func (r QueryResult) String() string {
	out, err := json.Marshal(r)
	if err != nil {
		logp.Warn("failed to marshal QueryResult (%v): %#v", err, r)
		return "ERROR"
	}
	return string(out)
}

func withQueryResult(status int, resp []byte, err error) (int, *QueryResult, error) {
	if err != nil {
		return status, nil, err
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
func (es *Connection) Index(
	index, docType, id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}
	return withQueryResult(es.apiCall(method, index, docType, id, "", params, body))
}

func (es *Connection) Ingest(
	index, docType, pipeline, id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}
	return withQueryResult(es.apiCall(method, index, docType, id, pipeline, params, body))
}

// Refresh an index. Call this after doing inserts or creating/deleting
// indexes in unit tests.
func (es *Connection) Refresh(index string) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("POST", index, "", "_refresh", "", nil, nil))
}

// CreateIndex creates a new index, optionally with settings and mappings passed in
// the body.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html
//
func (es *Connection) CreateIndex(index string, body interface{}) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("PUT", index, "", "", "", nil, body))
}

// Delete deletes a typed JSON document from a specific index based on its id.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-delete.html
func (es *Connection) Delete(index string, docType string, id string, params map[string]string) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("DELETE", index, docType, id, "", params, nil))
}

// CreatePipeline create a new ingest pipeline with name id.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/put-pipeline-api.html
func (es *Connection) CreatePipeline(
	id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("PUT", "_ingest", "pipeline", id, "", params, body))
}

// DeletePipeline deletes an ingest pipeline by id.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-pipeline-api.html
func (es *Connection) DeletePipeline(
	id string,
	params map[string]string,
) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("DELETE", "_ingest", "pipeline", id, "", params, nil))
}

// A search request can be executed purely using a URI by providing request parameters.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/search-uri-request.html
func (es *Connection) SearchURI(index string, docType string, params map[string]string) (int, *SearchResults, error) {
	status, resp, err := es.apiCall("GET", index, docType, "_search", "", params, nil)
	if err != nil {
		return status, nil, err
	}
	result, err := readSearchResult(resp)
	return status, result, err
}

func (es *Connection) CountSearchURI(
	index string, docType string,
	params map[string]string,
) (int, *CountResults, error) {
	status, resp, err := es.apiCall("GET", index, docType, "_count", "", params, nil)
	if err != nil {
		return status, nil, err
	}
	result, err := readCountResult(resp)
	return status, result, err
}

func (es *Connection) apiCall(
	method, index, docType, id, pipeline string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {
	path, err := makePath(index, docType, id)
	if err != nil {
		return 0, nil, err
	}
	return es.request(method, path, pipeline, params, body)
}
