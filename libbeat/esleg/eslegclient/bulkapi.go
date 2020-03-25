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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	ErrTempBulkFailure = errors.New("temporary bulk send failure")
)

type BulkIndexAction struct {
	Index BulkMeta `json:"index" struct:"index"`
}

type BulkCreateAction struct {
	Create BulkMeta `json:"create" struct:"create"`
}

type BulkMeta struct {
	Index    string `json:"_index" struct:"_index"`
	DocType  string `json:"_type,omitempty" struct:"_type,omitempty"`
	Pipeline string `json:"pipeline,omitempty" struct:"pipeline,omitempty"`
	ID       string `json:"_id,omitempty" struct:"_id,omitempty"`
}

type bulkRequest struct {
	requ *http.Request
}

// BulkResult contains the result of a bulk API request.
type BulkResult json.RawMessage

// Bulk performs many index/delete operations in a single API call.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-bulk.html
func (conn *Connection) Bulk(
	index, docType string,
	params map[string]string, body []interface{},
) (int, BulkResult, error) {
	if len(body) == 0 {
		return 0, nil, nil
	}

	enc := conn.Encoder
	enc.Reset()
	if err := bulkEncode(conn.log, enc, body); err != nil {
		return 0, nil, err
	}

	requ, err := newBulkRequest(conn.URL, index, docType, params, enc)
	if err != nil {
		return 0, nil, err
	}

	return conn.sendBulkRequest(requ)
}

// SendMonitoringBulk creates a HTTP request to the X-Pack Monitoring API containing a bunch of
// operations and sends them to Elasticsearch. The request is retransmitted up to max_retries
// before returning an error.
func (conn *Connection) SendMonitoringBulk(
	params map[string]string,
	body []interface{},
) (BulkResult, error) {
	if len(body) == 0 {
		return nil, nil
	}

	enc := conn.Encoder
	enc.Reset()
	if err := bulkEncode(conn.log, enc, body); err != nil {
		return nil, err
	}

	if !conn.version.IsValid() {
		if err := conn.Connect(); err != nil {
			return nil, err
		}
	}

	requ, err := newMonitoringBulkRequest(conn.GetVersion(), conn.URL, params, enc)
	if err != nil {
		return nil, err
	}

	_, result, err := conn.sendBulkRequest(requ)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func newBulkRequest(
	urlStr string,
	index, docType string,
	params map[string]string,
	body BodyEncoder,
) (*bulkRequest, error) {
	path, err := makePath(index, docType, "_bulk")
	if err != nil {
		return nil, err
	}

	return newBulkRequestWithPath(urlStr, path, params, body)
}

func newMonitoringBulkRequest(
	esVersion common.Version,
	urlStr string,
	params map[string]string,
	body BodyEncoder,
) (*bulkRequest, error) {
	var path string
	var err error
	if esVersion.Major < 7 {
		path, err = makePath("_xpack", "monitoring", "_bulk")
	} else {
		path, err = makePath("_monitoring", "bulk", "")
	}

	if err != nil {
		return nil, err
	}

	return newBulkRequestWithPath(urlStr, path, params, body)
}

func newBulkRequestWithPath(
	urlStr string,
	path string,
	params map[string]string,
	body BodyEncoder,
) (*bulkRequest, error) {
	url := addToURL(urlStr, path, "", params)

	var reader io.Reader
	if body != nil {
		reader = body.Reader()
	}

	requ, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		body.AddHeader(&requ.Header)
	}

	r := bulkRequest{
		requ: requ,
	}
	r.reset(body)

	return &r, nil
}

func (r *bulkRequest) reset(body BodyEncoder) {
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

	body.AddHeader(&r.requ.Header)
}

func (conn *Connection) sendBulkRequest(requ *bulkRequest) (int, BulkResult, error) {
	status, resp, err := conn.execHTTPRequest(requ.requ)
	return status, BulkResult(resp), err
}

func bulkEncode(log *logp.Logger, out BulkWriter, body []interface{}) error {
	for _, obj := range body {
		if err := out.AddRaw(obj); err != nil {
			log.Debugf("Failed to encode message: %s", err)
			return err
		}
	}
	return nil
}
