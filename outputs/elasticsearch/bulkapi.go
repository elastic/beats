package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

type EventMsg struct {
	Trans outputs.Transactioner
	Ts    time.Time
	Event common.MapStr
}

// Create a HTTP request containing a bunch of operations and send them to Elasticsearch.
// The request is retransmitted up to max_retries before returning an error.
func (es *Elasticsearch) BulkRequest(method string, path string,
	params map[string]string, body chan interface{}) ([]byte, error) {

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for obj := range body {
		enc.Encode(obj)
	}

	if buf.Len() == 0 {
		logp.Debug("elasticsearch", "Empty channel. Wait for more data.")
		return nil, nil
	}

	var errors []error

	for attempt := 0; attempt < es.MaxRetries; attempt++ {

		conn := es.connectionPool.GetConnection()
		logp.Debug("elasticsearch", "Use connection %s", conn.Url)

		url := conn.Url + path
		if len(params) > 0 {
			url = url + "?" + UrlEncode(params)
		}
		logp.Debug("elasticsearch", "Sending bulk request to %s", url)

		req, err := http.NewRequest(method, url, &buf)
		if err != nil {
			return nil, fmt.Errorf("NewRequest fails: %s", err)
		}

		resp, retry, err := es.PerformRequest(conn, req)
		if retry == true {
			// retry
			if err != nil {
				errors = append(errors, err)
			}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("PerformRequest fails: %s", err)
		}
		return resp, nil
	}

	logp.Warn("Request fails to be send after %d retries", es.MaxRetries)

	return nil, fmt.Errorf("Request fails after %d retries. Errors: %v", es.MaxRetries, errors)
}

// Perform many index/delete operations in a single API call.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html
func (es *Elasticsearch) Bulk(index string, doc_type string,
	params map[string]string, body chan interface{}) (*QueryResult, error) {

	path, err := MakePath(index, doc_type, "_bulk")
	if err != nil {
		return nil, err
	}

	resp, err := es.BulkRequest("POST", path, params, body)
	if err != nil {
		return nil, err
	}

	return ReadQueryResult(resp)
}
