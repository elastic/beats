package elasticsearch

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type bulkMeta struct {
	Index bulkMetaIndex `json:"index"`
}

type bulkMetaIndex struct {
	Index   string `json:"_index"`
	DocType string `json:"_type"`
}

// MetaBuilder creates meta data for bulk requests
type MetaBuilder func(interface{}) interface{}

type bulkRequest struct {
	requ *http.Request
}

type bulkBody interface {
	Reader() io.Reader
}

type bulkResult struct {
	raw []byte
}

type jsonBulkBody struct {
	buf *bytes.Buffer
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
		return nil, nil
	}

	bulkBody := newJSONBulkBody(nil)
	if err := bulkEncode(bulkBody, metaBuilder, body); err != nil {
		return nil, err
	}

	requ, err := newBulkRequest(conn.URL, index, docType, params, bulkBody)
	if err != nil {
		return nil, err
	}

	_, result, err := conn.sendBulkRequest(requ)
	if err != nil {
		return nil, err
	}
	return readQueryResult(result.raw)
}

func newBulkRequest(
	urlStr string,
	index, docType string,
	params map[string]string,
	body bulkBody,
) (*bulkRequest, error) {
	path, err := makePath(index, docType, "_bulk")
	if err != nil {
		return nil, err
	}

	url := makeURL(urlStr, path, params)

	var reader io.Reader
	if body != nil {
		reader = body.Reader()
	}

	requ, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, err
	}

	return &bulkRequest{
		requ: requ,
	}, nil
}

func (r *bulkRequest) Reset(body bulkBody) {
	bdy := body.Reader()

	rc, ok := bdy.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(bdy)
	}

	switch v := bdy.(type) {
	case *bytes.Buffer:
		r.requ.ContentLength = int64(v.Len())
	case *bytes.Reader:
		r.requ.ContentLength = int64(v.Len())
	case *strings.Reader:
		r.requ.ContentLength = int64(v.Len())
	}

	r.requ.Header = http.Header{}
	r.requ.Body = rc
}

func (conn *Connection) sendBulkRequest(requ *bulkRequest) (int, bulkResult, error) {
	status, resp, err := conn.execHTTPRequest(requ.requ)
	if err != nil {
		return status, bulkResult{}, err
	}

	result, err := readBulkResult(resp)
	return status, result, err
}

func readBulkResult(obj []byte) (bulkResult, error) {
	return bulkResult{obj}, nil
}

func newJSONBulkBody(buf *bytes.Buffer) *jsonBulkBody {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	return &jsonBulkBody{buf}
}

func (b *jsonBulkBody) Reset() {
	b.buf.Reset()
}

func (b *jsonBulkBody) Reader() io.Reader {
	return b.buf
}

func (b *jsonBulkBody) AddRaw(raw interface{}) error {
	enc := json.NewEncoder(b.buf)
	return enc.Encode(raw)
}

func (b *jsonBulkBody) Add(meta, obj interface{}) error {
	enc := json.NewEncoder(b.buf)
	pos := b.buf.Len()

	if err := enc.Encode(meta); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	if err := enc.Encode(obj); err != nil {
		b.buf.Truncate(pos)
		return err
	}
	return nil
}
func bulkEncode(out *jsonBulkBody, metaBuilder MetaBuilder, body []interface{}) error {
	if metaBuilder == nil {
		for _, obj := range body {
			if err := out.AddRaw(obj); err != nil {
				debug("Failed to encode message: %s", err)
				return err
			}
		}
	} else {
		for _, obj := range body {
			meta := metaBuilder(obj)
			if err := out.Add(meta, obj); err != nil {
				debug("Failed to encode message: %s", err)
			}
		}
	}
	return nil
}
