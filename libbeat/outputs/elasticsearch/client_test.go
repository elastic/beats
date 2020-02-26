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

// +build !integration

package elasticsearch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/outest"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/version"
)

func readStatusItem(in []byte) (int, string, error) {
	reader := NewJSONReader(in)
	code, msg, err := BulkReadItemStatus(reader)
	return code, string(msg), err
}

func TestESNoErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 200}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", msg)
}

func TestES1StyleErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 400, "error": "test error"}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `"test error"`, msg)
}

func TestES2StyleErrorStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 400, "error": {"reason": "test_error"}}}`)
	code, msg, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, `{"reason": "test_error"}`, msg)
}

func TestES2StyleExtendedErrorStatus(t *testing.T) {
	response := []byte(`
    {
      "create": {
        "status": 400,
        "error": {
          "reason": "test_error",
          "transient": false,
          "extra": null
        }
      }
    }`)
	code, _, err := readStatusItem(response)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
}

func TestCollectPublishFailsNone(t *testing.T) {
	N := 100
	item := `{"create": {"status": 200}},`
	response := []byte(`{"items": [` + strings.Repeat(item, N) + `]}`)

	event := common.MapStr{"field": 1}
	events := make([]publisher.Event, N)
	for i := 0; i < N; i++ {
		events[i] = publisher.Event{Content: beat.Event{Fields: event}}
	}

	reader := NewJSONReader(response)
	res, _ := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailMiddle(t *testing.T) {
	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event, eventFail, event}

	reader := NewJSONReader(response)
	res, stats := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		assert.Equal(t, eventFail, res[0])
	}
	assert.Equal(t, stats, bulkResultStats{acked: 2, fails: 1, tooMany: 1})
}

func TestCollectPublishFailAll(t *testing.T) {
	response := []byte(`
    { "items": [
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event, event, event}

	reader := NewJSONReader(response)
	res, stats := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
	assert.Equal(t, stats, bulkResultStats{fails: 3, tooMany: 3})
}

func TestCollectPipelinePublishFail(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	response := []byte(`{
      "took": 0, "ingest_took": 0, "errors": true,
      "items": [
        {
          "index": {
            "_index": "filebeat-2016.08.10",
            "_type": "log",
            "_id": null,
            "status": 500,
            "error": {
              "type": "exception",
              "reason": "java.lang.IllegalArgumentException: java.lang.IllegalArgumentException: field [fail_on_purpose] not present as part of path [fail_on_purpose]",
              "caused_by": {
                "type": "illegal_argument_exception",
                "reason": "java.lang.IllegalArgumentException: field [fail_on_purpose] not present as part of path [fail_on_purpose]",
                "caused_by": {
                  "type": "illegal_argument_exception",
                  "reason": "field [fail_on_purpose] not present as part of path [fail_on_purpose]"
                }
              },
              "header": {
                "processor_type": "lowercase"
              }
            }
          }
        }
      ]
    }`)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event}

	reader := NewJSONReader(response)
	res, _ := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, events, res)
}

func BenchmarkCollectPublishFailsNone(b *testing.B) {
	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 200}},
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 1}}}
	events := []publisher.Event{event, event, event}

	reader := NewJSONReader(nil)
	for i := 0; i < b.N; i++ {
		reader.init(response)
		res, _ := bulkCollectPublishFails(reader, events)
		if len(res) != 0 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailMiddle(b *testing.B) {
	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event, eventFail, event}

	reader := NewJSONReader(nil)
	for i := 0; i < b.N; i++ {
		reader.init(response)
		res, _ := bulkCollectPublishFails(reader, events)
		if len(res) != 1 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailAll(b *testing.B) {
	response := []byte(`
    { "items": [
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event, event, event}

	reader := NewJSONReader(nil)
	for i := 0; i < b.N; i++ {
		reader.init(response)
		res, _ := bulkCollectPublishFails(reader, events)
		if len(res) != 3 {
			b.Fail()
		}
	}
}

func TestClientWithHeaders(t *testing.T) {
	requestCount := 0
	// start a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "testing value", r.Header.Get("X-Test"))
		// from the documentation: https://golang.org/pkg/net/http/
		// For incoming requests, the Host header is promoted to the
		// Request.Host field and removed from the Header map.
		assert.Equal(t, "myhost.local", r.Host)

		bulkResponse := `{"items":[{"index":{}},{"index":{}},{"index":{}}]}`
		fmt.Fprintln(w, bulkResponse)
		requestCount++
	}))
	defer ts.Close()

	client, err := NewClient(ClientSettings{
		URL:   ts.URL,
		Index: outil.MakeSelector(outil.ConstSelectorExpr("test")),
		Headers: map[string]string{
			"host":   "myhost.local",
			"X-Test": "testing value",
		},
	}, nil)
	assert.NoError(t, err)

	// simple ping
	client.Ping()
	assert.Equal(t, 1, requestCount)

	// bulk request
	event := beat.Event{Fields: common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "libbeat",
		"message":    "Test message from libbeat",
	}}

	batch := outest.NewBatch(event, event, event)
	err = client.Publish(batch)
	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount)
}

func TestAddToURL(t *testing.T) {
	type Test struct {
		url      string
		path     string
		pipeline string
		params   map[string]string
		expected string
	}
	tests := []Test{
		{
			url:      "localhost:9200",
			path:     "/path",
			pipeline: "",
			params:   make(map[string]string),
			expected: "localhost:9200/path",
		},
		{
			url:      "localhost:9200/",
			path:     "/path",
			pipeline: "",
			params:   make(map[string]string),
			expected: "localhost:9200/path",
		},
		{
			url:      "localhost:9200",
			path:     "/path",
			pipeline: "pipeline_1",
			params:   make(map[string]string),
			expected: "localhost:9200/path?pipeline=pipeline_1",
		},
		{
			url:      "localhost:9200/",
			path:     "/path",
			pipeline: "",
			params: map[string]string{
				"param": "value",
			},
			expected: "localhost:9200/path?param=value",
		},
	}
	for _, test := range tests {
		url := addToURL(test.url, test.path, test.pipeline, test.params)
		assert.Equal(t, url, test.expected)
	}
}

type testBulkRecorder struct {
	data     []interface{}
	inAction bool
}

func TestBulkEncodeEvents(t *testing.T) {
	cases := map[string]struct {
		docType string
		config  common.MapStr
		events  []common.MapStr
	}{
		"Beats 7.x event": {
			docType: "doc",
			config:  common.MapStr{},
			events:  []common.MapStr{{"message": "test"}},
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(test.config)
			info := beat.Info{
				IndexPrefix: "test",
				Version:     version.GetDefaultVersion(),
			}

			im, err := idxmgmt.DefaultSupport(nil, info, common.NewConfig())
			require.NoError(t, err)

			index, pipeline, err := buildSelectors(im, info, cfg)
			require.NoError(t, err)

			events := make([]publisher.Event, len(test.events))
			for i, fields := range test.events {
				events[i] = publisher.Event{
					Content: beat.Event{
						Timestamp: time.Now(),
						Fields:    fields,
					},
				}
			}

			recorder := &testBulkRecorder{}

			encoded := bulkEncodePublishRequest(common.Version{Major: 7, Minor: 5}, recorder, index, pipeline, test.docType, events)
			assert.Equal(t, len(events), len(encoded), "all events should have been encoded")
			assert.False(t, recorder.inAction, "incomplete bulk")

			// check meta-data for each event
			for i := 0; i < len(recorder.data); i += 2 {
				var meta bulkEventMeta
				switch v := recorder.data[i].(type) {
				case bulkCreateAction:
					meta = v.Create
				case bulkIndexAction:
					meta = v.Index
				default:
					panic("unknown type")
				}

				assert.NotEqual(t, "", meta.Index)
				assert.Equal(t, test.docType, meta.DocType)
			}

			// TODO: customer per test case validation
		})
	}
}

func (r *testBulkRecorder) Add(meta, obj interface{}) error {
	if r.inAction {
		panic("can not add a new action if other action is active")
	}

	r.data = append(r.data, meta, obj)
	return nil
}

func (r *testBulkRecorder) AddRaw(raw interface{}) error {
	r.data = append(r.data)
	r.inAction = !r.inAction
	return nil
}

func TestClientWithAPIKey(t *testing.T) {
	var headers http.Header

	// Start a mock HTTP server, save request headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
	}))
	defer ts.Close()

	client, err := NewClient(ClientSettings{
		URL:    ts.URL,
		APIKey: "hyokHG4BfWk5viKZ172X:o45JUkyuS--yiSAuuxl8Uw",
	}, nil)
	assert.NoError(t, err)

	client.Ping()
	assert.Equal(t, "ApiKey aHlva0hHNEJmV2s1dmlLWjE3Mlg6bzQ1SlVreXVTLS15aVNBdXV4bDhVdw==", headers.Get("Authorization"))
}

func TestBulkReadToItems(t *testing.T) {
	response := []byte(`{
		"errors": false,
		"items": [
			{"create": {"status": 200}},
			{"create": {"status": 300}},
			{"create": {"status": 400}}
    ]}`)

	reader := NewJSONReader(response)

	err := BulkReadToItems(reader)
	assert.NoError(t, err)

	for status := 200; status <= 400; status += 100 {
		err = reader.ExpectDict()
		assert.NoError(t, err)

		kind, raw, err := reader.nextFieldName()
		assert.NoError(t, err)
		assert.Equal(t, mapKeyEntity, kind)
		assert.Equal(t, []byte("create"), raw)

		err = reader.ExpectDict()
		assert.NoError(t, err)

		kind, raw, err = reader.nextFieldName()
		assert.NoError(t, err)
		assert.Equal(t, mapKeyEntity, kind)
		assert.Equal(t, []byte("status"), raw)

		code, err := reader.nextInt()
		assert.NoError(t, err)
		assert.Equal(t, status, code)

		_, _, err = reader.endDict()
		assert.NoError(t, err)

		_, _, err = reader.endDict()
		assert.NoError(t, err)
	}
}

func TestBulkReadItemStatus(t *testing.T) {
	response := []byte(`{"create": {"status": 200}}`)

	reader := NewJSONReader(response)
	code, _, err := BulkReadItemStatus(reader)
	assert.NoError(t, err)
	assert.Equal(t, 200, code)
}
