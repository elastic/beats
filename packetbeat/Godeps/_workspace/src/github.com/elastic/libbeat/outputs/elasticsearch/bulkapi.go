package elasticsearch

import (
	"bytes"
	"encoding/json"

	"github.com/elastic/libbeat/logp"
)

// MetaBuilder creates meta data for bulk requests
type MetaBuilder func(interface{}) interface{}

type bulkRequest struct {
	es     *Connection
	buf    bytes.Buffer
	enc    *json.Encoder
	path   string
	params map[string]string
}

type bulkMeta struct {
	Index bulkMetaIndex `json:"index"`
}

type bulkMetaIndex struct {
	Index   string `json:"_index"`
	DocType string `json:"_type"`
}

type BulkResult struct {
	Items []json.RawMessage `json:"items"`
}

func (r *bulkRequest) Send(meta, obj interface{}) error {
	var err error

	pos := r.buf.Len()
	if err = r.enc.Encode(meta); err != nil {
		return err
	}
	if err = r.enc.Encode(obj); err != nil {
		r.buf.Truncate(pos) // remove meta object from buffer
	}
	return err
}

func (r *bulkRequest) Flush() (int, *BulkResult, error) {
	if r.buf.Len() == 0 {
		logp.Debug("elasticsearch", "Empty channel. Wait for more data.")
		return 0, nil, nil
	}

	status, resp, err := r.es.sendBulkRequest("POST", r.path, r.params, &r.buf)
	if err != nil {
		return status, nil, err
	}
	r.buf.Truncate(0)

	result, err := readBulkResult(resp)
	return status, result, err
}

// Bulk performs many index/delete operations in a single API call.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html
func (conn *Connection) Bulk(
	index, docType string,
	params map[string]string, body []interface{},
) (*QueryResult, error) {
	return conn.BulkWith(index, docType, params, nil, body)
}

// BulkWith creates a HTTP request containing a bunch of operations and send
// them to Elasticsearch. The request is retransmitted up to max_retries before
// returning an error.
func (conn *Connection) BulkWith(
	index string,
	docType string,
	params map[string]string,
	metaBuilder MetaBuilder,
	body []interface{},
) (*QueryResult, error) {
	if len(body) == 0 {
		logp.Debug("elasticsearch", "Empty channel. Wait for more data.")
		return nil, nil
	}

	path, err := makePath(index, docType, "_bulk")
	if err != nil {
		return nil, err
	}

	buf := bulkEncode(metaBuilder, body)
	if buf.Len() == 0 {
		logp.Debug("elasticsearch", "Empty channel. Wait for more data.")
		return nil, nil
	}

	_, resp, err := conn.sendBulkRequest("POST", path, params, &buf)
	if err != nil {
		return nil, err
	}
	return readQueryResult(resp)
}

func (conn *Connection) startBulkRequest(
	index string,
	docType string,
	params map[string]string,
) (*bulkRequest, error) {
	path, err := makePath(index, docType, "_bulk")
	if err != nil {
		return nil, err
	}

	r := &bulkRequest{
		es:     conn,
		path:   path,
		params: params,
	}
	r.enc = json.NewEncoder(&r.buf)
	return r, nil
}

func (conn *Connection) sendBulkRequest(
	method, path string,
	params map[string]string,
	buf *bytes.Buffer,
) (int, []byte, error) {
	url := makeURL(conn.URL, path, params)
	logp.Debug("elasticsearch", "Sending bulk request to %s", url)

	return conn.execRequest(method, url, buf)
}

func bulkEncode(metaBuilder MetaBuilder, body []interface{}) bytes.Buffer {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if metaBuilder == nil {
		for _, obj := range body {
			pos := buf.Len()
			if err := enc.Encode(obj); err != nil {
				debug("Failed to encode message: %s", err)
				buf.Truncate(pos)
			}
		}
	} else {
		for _, obj := range body {
			pos := buf.Len()
			meta := metaBuilder(obj)
			err := enc.Encode(meta)
			if err == nil {
				err = enc.Encode(obj)
			}
			if err != nil {
				debug("Failed to encode message: %s", err)
				buf.Truncate(pos)
			}
		}
	}
	return buf
}

func readBulkResult(obj []byte) (*BulkResult, error) {
	if obj == nil {
		return nil, nil
	}

	var result BulkResult
	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
