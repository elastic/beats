package elasticsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Elasticsearch struct {
	Url string

	client *http.Client
}

type ESSearchResults struct {
	Took   int                        `json:"took"`
	Shards json.RawMessage            `json:"_shards"`
	Hits   ESHits                     `json:"hits"`
	Aggs   map[string]json.RawMessage `json:"aggregations"`
}

type ESHits struct {
	Total int
	Hits  []json.RawMessage `json:"hits"`
}

func NewElasticsearch(url string) *Elasticsearch {
	if len(url) == 0 {
		url = "http://localhost:9200"
	}
	return &Elasticsearch{
		Url:    url,
		client: &http.Client{},
	}
}

// Generic request method. Returns the HTTP response that we get from ES.
// If ES returns an error HTTP code (>299), the error is non-nil and the
// response is also non-nil.
func (es *Elasticsearch) Request(method string, index string, path string,
	data io.Reader) (*http.Response, error) {

	url := fmt.Sprintf("%s/%s/%s", es.Url, index, path)

	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, err
	}

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return resp, fmt.Errorf("ES returned an error: %s", resp.Status)
	}

	return resp, nil
}

// Refresh an index. Call this after doing inserts or creating/deleting
// indexes in unit tests.
func (es *Elasticsearch) Refresh(index string) (*http.Response, error) {
	return es.Request("POST", index, "_refresh", nil)
}

func (es *Elasticsearch) DeleteIndex(index string) (*http.Response, error) {
	path := fmt.Sprintf("%s/%s", es.Url, index)

	req, err := http.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (es *Elasticsearch) Search(index string, params string, reqjson string) (*http.Response, error) {

	path := fmt.Sprintf("%s/%s/_search%s", es.Url, index, params)

	req, err := http.NewRequest("GET", path, strings.NewReader(reqjson))
	if err != nil {
		return nil, err
	}

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return resp, fmt.Errorf("ES returned an error: %s", resp.Status)
	}

	return resp, nil
}
