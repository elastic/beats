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

//go:build !integration
// +build !integration

package elasticsearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/beat"
	e "github.com/menderesk/beats/v7/libbeat/beat/events"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/esleg/eslegclient"
	"github.com/menderesk/beats/v7/libbeat/idxmgmt"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/outputs/outest"
	"github.com/menderesk/beats/v7/libbeat/outputs/outil"
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/version"
)

type testIndexSelector struct{}

func (testIndexSelector) Select(event *beat.Event) (string, error) {
	return "test", nil
}

type batchMock struct {
	// we embed the interface so we are able to implement the interface partially,
	// only functions needed for tests are implemented
	// if you use a function that is not implemented in the mock it will panic
	publisher.Batch
	events      []publisher.Event
	ack         bool
	drop        bool
	retryEvents []publisher.Event
}

func (bm batchMock) Events() []publisher.Event {
	return bm.events
}
func (bm *batchMock) ACK() {
	bm.ack = true
}
func (bm *batchMock) Drop() {
	bm.drop = true
}
func (bm *batchMock) RetryEvents(events []publisher.Event) {
	bm.retryEvents = events
}

func TestPublishStatusCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 1}}}
	events := []publisher.Event{event}

	t.Run("returns pre-defined error and drops batch when 413", func(t *testing.T) {
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
		}))
		defer esMock.Close()

		client, err := NewClient(
			ClientSettings{
				ConnectionSettings: eslegclient.ConnectionSettings{
					URL: esMock.URL,
				},
				Index: testIndexSelector{},
			},
			nil,
		)
		assert.NoError(t, err)

		event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 1}}}
		events := []publisher.Event{event}
		batch := &batchMock{
			events: events,
		}

		err = client.Publish(ctx, batch)

		assert.Error(t, err)
		assert.Equal(t, errPayloadTooLarge, err, "should be a pre-defined error")
		assert.True(t, batch.drop, "should must be dropped")
	})

	t.Run("retries the batch if bad HTTP status", func(t *testing.T) {
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer esMock.Close()

		client, err := NewClient(
			ClientSettings{
				ConnectionSettings: eslegclient.ConnectionSettings{
					URL: esMock.URL,
				},
				Index: testIndexSelector{},
			},
			nil,
		)
		assert.NoError(t, err)

		batch := &batchMock{
			events: events,
		}

		err = client.Publish(ctx, batch)

		assert.Error(t, err)
		assert.False(t, batch.ack, "should not be acknowledged")
		assert.Len(t, batch.retryEvents, len(events), "all events should be in retry")
	})
}

func TestCollectPublishFailsNone(t *testing.T) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(t, err)

	N := 100
	item := `{"create": {"status": 200}},`
	response := []byte(`{"items": [` + strings.Repeat(item, N) + `]}`)

	event := common.MapStr{"field": 1}
	events := make([]publisher.Event, N)
	for i := 0; i < N; i++ {
		events[i] = publisher.Event{Content: beat.Event{Fields: event}}
	}

	res, _ := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailMiddle(t *testing.T) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(t, err)

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

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		assert.Equal(t, eventFail, res[0])
	}
	assert.Equal(t, bulkResultStats{acked: 2, fails: 1, tooMany: 1}, stats)
}

func TestCollectPublishFailDeadLetterQueue(t *testing.T) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "dead_letter_index",
		},
		nil,
	)
	assert.NoError(t, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {
		  "error" : {
			"root_cause" : [
			  {
				"type" : "mapper_parsing_exception",
				"reason" : "failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'"
			  }
			],
			"type" : "mapper_parsing_exception",
			"reason" : "failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'",
			"caused_by" : {
			  "type" : "illegal_argument_exception",
			  "reason" : "For input string: \"bar1\""
			}
		  },
		  "status" : 400
		}
      },
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"bar": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: common.MapStr{"bar": "bar1"}}}
	events := []publisher.Event{event, eventFail, event}

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		expected := publisher.Event{
			Content: beat.Event{
				Fields: common.MapStr{
					"message":       "{\"bar\":\"bar1\"}",
					"error.type":    400,
					"error.message": "{\n\t\t\t\"root_cause\" : [\n\t\t\t  {\n\t\t\t\t\"type\" : \"mapper_parsing_exception\",\n\t\t\t\t\"reason\" : \"failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'\"\n\t\t\t  }\n\t\t\t],\n\t\t\t\"type\" : \"mapper_parsing_exception\",\n\t\t\t\"reason\" : \"failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'\",\n\t\t\t\"caused_by\" : {\n\t\t\t  \"type\" : \"illegal_argument_exception\",\n\t\t\t  \"reason\" : \"For input string: \\\"bar1\\\"\"\n\t\t\t}\n\t\t  }",
				},
				Meta: common.MapStr{
					dead_letter_marker_field: true,
				},
			},
		}
		assert.Equal(t, expected, res[0])
	}
	assert.Equal(t, bulkResultStats{acked: 2, fails: 1, nonIndexable: 0}, stats)
}

func TestCollectPublishFailDrop(t *testing.T) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(t, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {
		  "error" : {
			"root_cause" : [
			  {
				"type" : "mapper_parsing_exception",
				"reason" : "failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'"
			  }
			],
			"type" : "mapper_parsing_exception",
			"reason" : "failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'",
			"caused_by" : {
			  "type" : "illegal_argument_exception",
			  "reason" : "For input string: \"bar1\""
			}
		  },
		  "status" : 400
		}
      },
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"bar": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: common.MapStr{"bar": "bar1"}}}
	events := []publisher.Event{event, eventFail, event}

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 0, len(res))
	assert.Equal(t, bulkResultStats{acked: 2, fails: 0, nonIndexable: 1}, stats)
}

func TestCollectPublishFailAll(t *testing.T) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(t, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}},
      {"create": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event, event, event}

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
	assert.Equal(t, stats, bulkResultStats{fails: 3, tooMany: 3})
}

func TestCollectPipelinePublishFail(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(t, err)

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

	res, _ := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, events, res)
}

func BenchmarkCollectPublishFailsNone(b *testing.B) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(b, err)

	response := []byte(`
    { "items": [
      {"create": {"status": 200}},
      {"create": {"status": 200}},
      {"create": {"status": 200}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 1}}}
	events := []publisher.Event{event, event, event}

	for i := 0; i < b.N; i++ {
		res, _ := client.bulkCollectPublishFails(response, events)
		if len(res) != 0 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailMiddle(b *testing.B) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(b, err)

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

	for i := 0; i < b.N; i++ {
		res, _ := client.bulkCollectPublishFails(response, events)
		if len(res) != 1 {
			b.Fail()
		}
	}
}

func BenchmarkCollectPublishFailAll(b *testing.B) {
	client, err := NewClient(
		ClientSettings{
			NonIndexableAction: "drop",
		},
		nil,
	)
	assert.NoError(b, err)

	response := []byte(`
    { "items": [
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}},
      {"creatMiddlee": {"status": 429, "error": "ups"}}
    ]}
  `)

	event := publisher.Event{Content: beat.Event{Fields: common.MapStr{"field": 2}}}
	events := []publisher.Event{event, event, event}

	for i := 0; i < b.N; i++ {
		res, _ := client.bulkCollectPublishFails(response, events)
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

		var response string
		if r.URL.Path == "/" {
			response = `{ "version": { "number": "7.6.0" } }`
		} else {
			response = `{"items":[{"index":{}},{"index":{}},{"index":{}}]}`

		}
		fmt.Fprintln(w, response)
		requestCount++
	}))
	defer ts.Close()

	client, err := NewClient(ClientSettings{
		ConnectionSettings: eslegclient.ConnectionSettings{
			URL: ts.URL,
			Headers: map[string]string{
				"host":   "myhost.local",
				"X-Test": "testing value",
			},
		},
		Index: outil.MakeSelector(outil.ConstSelectorExpr("test", outil.SelectorLowerCase)),
	}, nil)
	assert.NoError(t, err)

	// simple ping
	client.Connect()
	assert.Equal(t, 1, requestCount)

	// bulk request
	event := beat.Event{Fields: common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "libbeat",
		"message":    "Test message from libbeat",
	}}

	batch := outest.NewBatch(event, event, event)
	err = client.Publish(context.Background(), batch)
	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount)
}

func TestBulkEncodeEvents(t *testing.T) {
	cases := map[string]struct {
		version string
		docType string
		config  common.MapStr
		events  []common.MapStr
	}{
		"6.x": {
			version: "6.8.0",
			docType: "doc",
			config:  common.MapStr{},
			events:  []common.MapStr{{"message": "test"}},
		},
		"latest": {
			version: version.GetDefaultVersion(),
			docType: "",
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
				Version:     test.version,
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

			client, err := NewClient(
				ClientSettings{
					Index:    index,
					Pipeline: pipeline,
				},
				nil,
			)
			assert.NoError(t, err)

			encoded, bulkItems := client.bulkEncodePublishRequest(*common.MustNewVersion(test.version), events)
			assert.Equal(t, len(events), len(encoded), "all events should have been encoded")
			assert.Equal(t, 2*len(events), len(bulkItems), "incomplete bulk")

			// check meta-data for each event
			for i := 0; i < len(bulkItems); i += 2 {
				var meta eslegclient.BulkMeta
				switch v := bulkItems[i].(type) {
				case eslegclient.BulkCreateAction:
					meta = v.Create
				case eslegclient.BulkIndexAction:
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

func TestBulkEncodeEventsWithOpType(t *testing.T) {
	cases := []common.MapStr{
		{"_id": "111", "op_type": e.OpTypeIndex, "message": "test 1", "bulkIndex": 0},
		{"_id": "112", "message": "test 2", "bulkIndex": 2},
		{"_id": "", "op_type": e.OpTypeDelete, "message": "test 6", "bulkIndex": -1}, // this won't get encoded due to missing _id
		{"_id": "", "message": "test 3", "bulkIndex": 4},
		{"_id": "114", "op_type": e.OpTypeDelete, "message": "test 4", "bulkIndex": 6},
		{"_id": "115", "op_type": e.OpTypeIndex, "message": "test 5", "bulkIndex": 7},
	}

	cfg := common.MustNewConfigFrom(common.MapStr{})
	info := beat.Info{
		IndexPrefix: "test",
		Version:     version.GetDefaultVersion(),
	}

	im, err := idxmgmt.DefaultSupport(nil, info, common.NewConfig())
	require.NoError(t, err)

	index, pipeline, err := buildSelectors(im, info, cfg)
	require.NoError(t, err)

	events := make([]publisher.Event, len(cases))
	for i, fields := range cases {
		meta := common.MapStr{
			"_id": fields["_id"],
		}
		if opType, exists := fields["op_type"]; exists {
			meta[e.FieldMetaOpType] = opType
		}

		events[i] = publisher.Event{
			Content: beat.Event{
				Meta: meta,
				Fields: common.MapStr{
					"message": fields["message"],
				},
			},
		}
	}

	client, err := NewClient(
		ClientSettings{
			Index:    index,
			Pipeline: pipeline,
		},
		nil,
	)

	encoded, bulkItems := client.bulkEncodePublishRequest(*common.MustNewVersion(version.GetDefaultVersion()), events)
	require.Equal(t, len(events)-1, len(encoded), "all events should have been encoded")
	require.Equal(t, 9, len(bulkItems), "incomplete bulk")

	for i := 0; i < len(cases); i++ {
		bulkEventIndex, _ := cases[i]["bulkIndex"].(int)
		if bulkEventIndex == -1 {
			continue
		}
		caseOpType, _ := cases[i]["op_type"]
		caseMessage, _ := cases[i]["message"].(string)
		switch bulkItems[bulkEventIndex].(type) {
		case eslegclient.BulkCreateAction:
			validOpTypes := []interface{}{e.OpTypeCreate, nil}
			require.Contains(t, validOpTypes, caseOpType, caseMessage)
		case eslegclient.BulkIndexAction:
			require.Equal(t, e.OpTypeIndex, caseOpType, caseMessage)
		case eslegclient.BulkDeleteAction:
			require.Equal(t, e.OpTypeDelete, caseOpType, caseMessage)
		default:
			require.FailNow(t, "unknown type")
		}
	}

}

func TestClientWithAPIKey(t *testing.T) {
	var headers http.Header

	// Start a mock HTTP server, save request headers
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
	}))
	defer ts.Close()

	client, err := NewClient(ClientSettings{
		ConnectionSettings: eslegclient.ConnectionSettings{
			URL:    ts.URL,
			APIKey: "hyokHG4BfWk5viKZ172X:o45JUkyuS--yiSAuuxl8Uw",
		},
	}, nil)
	assert.NoError(t, err)

	client.Connect()
	assert.Equal(t, "ApiKey aHlva0hHNEJmV2s1dmlLWjE3Mlg6bzQ1SlVreXVTLS15aVNBdXV4bDhVdw==", headers.Get("Authorization"))
}
