package elasticsearch

import "encoding/json"

type QueryResult struct {
	Ok      bool            `json:"ok"`
	Index   string          `json:"_index"`
	Type    string          `json:"_type"`
	ID      string          `json:"_id"`
	Source  json.RawMessage `json:"_source"`
	Version int             `json:"_version"`
	Found   bool            `json:"found"`
	Exists  bool            `json:"exists"`
	Created bool            `json:"created"`
	Matches []string        `json:"matches"`
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
		return "ERROR"
	}
	return string(out)
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
func (es *Client) Index(
	index, docType, id string,
	params map[string]string,
	body interface{},
) (*QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}

	resp, err := es.apiCall(method, index, docType, id, params, body)
	if err != nil {
		return nil, err
	}
	return readQueryResult(resp)
}

// Refresh an index. Call this after doing inserts or creating/deleting
// indexes in unit tests.
func (es *Client) Refresh(index string) (*QueryResult, error) {
	resp, err := es.apiCall("POST", index, "", "_refresh", nil, nil)
	if err != nil {
		return nil, err
	}
	return readQueryResult(resp)
}

// CreateIndex creates a new index, optionally with settings and mappings passed in
// the body.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html
//
func (es *Client) CreateIndex(index string, body interface{}) (*QueryResult, error) {
	resp, err := es.apiCall("PUT", index, "", "", nil, body)
	if err != nil {
		return nil, err
	}
	return readQueryResult(resp)
}

// Delete deletes a typed JSON document from a specific index based on its id.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-delete.html
func (es *Client) Delete(index string, docType string, id string, params map[string]string) (*QueryResult, error) {
	resp, err := es.apiCall("DELETE", index, docType, id, params, nil)
	if err != nil {
		return nil, err
	}
	return readQueryResult(resp)
}

// A search request can be executed purely using a URI by providing request parameters.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/search-uri-request.html
func (es *Client) SearchURI(index string, docType string, params map[string]string) (*SearchResults, error) {
	resp, err := es.apiCall("GET", index, docType, "_search", params, nil)
	if err != nil {
		return nil, err
	}
	return readSearchResult(resp)
}

func (es *Client) CountSearchURI(
	index string, docType string,
	params map[string]string,
) (*CountResults, error) {
	resp, err := es.apiCall("GET", index, docType, "_count", params, nil)
	if err != nil {
		return nil, err
	}
	return readCountResult(resp)
}

func (es *Client) apiCall(
	method, index, docType, id string,
	params map[string]string,
	body interface{},
) ([]byte, error) {
	path, err := makePath(index, docType, id)
	if err != nil {
		return nil, err
	}
	return es.request(method, path, params, body)
}
