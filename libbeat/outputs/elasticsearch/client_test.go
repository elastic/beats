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

package elasticsearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	e "github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/version"
	c "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

type testIndexSelector struct{}

func (testIndexSelector) Select(event *beat.Event) (string, error) {
	return "test", nil
}

type batchMock struct {
	events      []publisher.Event
	ack         bool
	drop        bool
	canSplit    bool
	didSplit    bool
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
func (bm *batchMock) Retry()       { panic("unimplemented") }
func (bm *batchMock) Cancelled()   { panic("unimplemented") }
func (bm *batchMock) FreeEntries() {}
func (bm *batchMock) SplitRetry() bool {
	if bm.canSplit {
		bm.didSplit = true
	}
	return bm.canSplit
}
func (bm *batchMock) RetryEvents(events []publisher.Event) {
	bm.retryEvents = events
}

func TestPublish(t *testing.T) {
	makePublishTestClient := func(t *testing.T, url string) *Client {
		client, err := NewClient(
			clientSettings{
				observer:      outputs.NewNilObserver(),
				connection:    eslegclient.ConnectionSettings{URL: url},
				indexSelector: testIndexSelector{},
			},
			nil,
		)
		require.NoError(t, err)
		return client
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event1 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
	event2 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	event3 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 3}}}

	t.Run("splits large batches on status code 413", func(t *testing.T) {
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL)

		// Try publishing a batch that can be split
		batch := &batchMock{
			events:   []publisher.Event{event1},
			canSplit: true,
		}
		err := client.Publish(ctx, batch)

		assert.NoError(t, err, "Publish should split the batch without error")
		assert.True(t, batch.didSplit, "batch should be split")

		// Try publishing a batch that cannot be split
		batch = &batchMock{
			events:   []publisher.Event{event1},
			canSplit: false,
		}
		err = client.Publish(ctx, batch)

		assert.NoError(t, err, "Publish should drop the batch without error")
		assert.False(t, batch.didSplit, "batch should not be split")
		assert.True(t, batch.drop, "unsplittable batch should be dropped")
	})

	t.Run("retries the batch if bad HTTP status", func(t *testing.T) {
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL)

		batch := &batchMock{
			events: []publisher.Event{event1, event2},
		}

		err := client.Publish(ctx, batch)

		assert.Error(t, err)
		assert.False(t, batch.ack, "should not be acknowledged")
		assert.Len(t, batch.retryEvents, 2, "all events should be retried")
	})

	t.Run("live batches, still too big after split", func(t *testing.T) {
		// Test a live (non-mocked) batch where both events by themselves are
		// rejected by the server as too large after the initial split.
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL)

		// Because our tests don't use a live eventConsumer routine,
		// everything will happen synchronously and it's safe to track
		// test results directly without atomics/mutexes.
		done := false
		retryCount := 0
		batch := pipeline.NewBatchForTesting(
			[]publisher.Event{event1, event2, event3},
			func(b publisher.Batch) {
				// The retry function sends the batch back through Publish.
				// In a live pipeline it would instead be sent to eventConsumer
				// first and then back to Publish when an output worker was
				// available.
				retryCount++
				err := client.Publish(ctx, b)
				assert.NoError(t, err, "Publish should return without error")
			},
			func() { done = true },
		)
		err := client.Publish(ctx, batch)
		assert.NoError(t, err, "Publish should return without error")

		// For three events there should be four retries in total:
		// {[event1], [event2, event3]}, then {[event2], [event3]}.
		// "done" should be true because after splitting into individual
		// events, all 3 will fail and be dropped.
		assert.Equal(t, 4, retryCount, "3-event batch should produce 4 total retries")
		assert.True(t, done, "batch should be marked as done")
	})

	t.Run("live batches, one event too big after split", func(t *testing.T) {
		// Test a live (non-mocked) batch where a single event is too large
		// for the server to ingest but the others are ok.
		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			body := string(b)
			// Reject the batch as too large only if it contains event1
			if strings.Contains(body, "\"field\":1") {
				// Report batch too large
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = w.Write([]byte("Request failed to get to the server (status code: 413)")) // actual response from ES
			} else {
				// Report success with no events dropped
				w.WriteHeader(200)
				_, _ = io.WriteString(w, "{\"items\": []}")
			}
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL)

		// Because our tests don't use a live eventConsumer routine,
		// everything will happen synchronously and it's safe to track
		// test results directly without atomics/mutexes.
		done := false
		retryCount := 0
		batch := pipeline.NewBatchForTesting(
			[]publisher.Event{event1, event2, event3},
			func(b publisher.Batch) {
				// The retry function sends the batch back through Publish.
				// In a live pipeline it would instead be sent to eventConsumer
				// first and then back to Publish when an output worker was
				// available.
				retryCount++
				err := client.Publish(ctx, b)
				assert.NoError(t, err, "Publish should return without error")
			},
			func() { done = true },
		)
		err := client.Publish(ctx, batch)
		assert.NoError(t, err, "Publish should return without error")

		// There should be two retries: {[event1], [event2, event3]}.
		// The first split batch should fail and be dropped since it contains
		// event1, the other one should succeed.
		// "done" should be true because both split batches are completed
		// (one with failure, one with success).
		assert.Equal(t, 2, retryCount, "splitting with one large event should produce two retries")
		assert.True(t, done, "batch should be marked as done")
	})
}

func TestCollectPublishFailsNone(t *testing.T) {
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
		},
		nil,
	)
	assert.NoError(t, err)

	N := 100
	item := `{"create": {"status": 200}},`
	response := []byte(`{"items": [` + strings.Repeat(item, N) + `]}`)

	event := mapstr.M{"field": 1}
	events := make([]publisher.Event, N)
	for i := 0; i < N; i++ {
		events[i] = publisher.Event{Content: beat.Event{Fields: event}}
	}

	res, _ := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 0, len(res))
}

func TestCollectPublishFailMiddle(t *testing.T) {
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
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
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: "test_index",
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": "bar1"}}}
	events := []publisher.Event{event, eventFail, event}

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		expected := publisher.Event{
			Content: beat.Event{
				Fields: mapstr.M{
					"message":       "{\"bar\":\"bar1\"}",
					"error.type":    400,
					"error.message": "{\n\t\t\t\"root_cause\" : [\n\t\t\t  {\n\t\t\t\t\"type\" : \"mapper_parsing_exception\",\n\t\t\t\t\"reason\" : \"failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'\"\n\t\t\t  }\n\t\t\t],\n\t\t\t\"type\" : \"mapper_parsing_exception\",\n\t\t\t\"reason\" : \"failed to parse field [bar] of type [long] in document with id '1'. Preview of field's value: 'bar1'\",\n\t\t\t\"caused_by\" : {\n\t\t\t  \"type\" : \"illegal_argument_exception\",\n\t\t\t  \"reason\" : \"For input string: \\\"bar1\\\"\"\n\t\t\t}\n\t\t  }",
				},
				Meta: mapstr.M{
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
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: "",
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: mapstr.M{"bar": "bar1"}}}
	events := []publisher.Event{event, eventFail, event}

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 0, len(res))
	assert.Equal(t, bulkResultStats{acked: 2, fails: 0, nonIndexable: 1}, stats)
}

func TestCollectPublishFailAll(t *testing.T) {
	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	events := []publisher.Event{event, event, event}

	res, stats := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
	assert.Equal(t, stats, bulkResultStats{fails: 3, tooMany: 3})
}

func TestCollectPipelinePublishFail(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client, err := NewClient(
		clientSettings{
			observer: outputs.NewNilObserver(),
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
	events := []publisher.Event{event}

	res, _ := client.bulkCollectPublishFails(response, events)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, events, res)
}

func BenchmarkCollectPublishFailsNone(b *testing.B) {
	client, err := NewClient(
		clientSettings{
			observer:        outputs.NewNilObserver(),
			deadLetterIndex: "",
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
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
		clientSettings{
			observer: outputs.NewNilObserver(),
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}
	eventFail := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
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
		clientSettings{
			observer: outputs.NewNilObserver(),
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

	event := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 2}}}
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

	client, err := NewClient(clientSettings{
		observer: outputs.NewNilObserver(),
		connection: eslegclient.ConnectionSettings{
			URL: ts.URL,
			Headers: map[string]string{
				"host":   "myhost.local",
				"X-Test": "testing value",
			},
		},
		indexSelector: outil.MakeSelector(outil.ConstSelectorExpr("test", outil.SelectorLowerCase)),
	}, nil)
	assert.NoError(t, err)

	// simple ping
	err = client.Connect()
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// bulk request
	event := beat.Event{Fields: mapstr.M{
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
		config  mapstr.M
		events  []mapstr.M
	}{
		"6.x": {
			version: "6.8.0",
			docType: "doc",
			config:  mapstr.M{},
			events:  []mapstr.M{{"message": "test"}},
		},
		"latest": {
			version: version.GetDefaultVersion(),
			docType: "",
			config:  mapstr.M{},
			events:  []mapstr.M{{"message": "test"}},
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := c.MustNewConfigFrom(test.config)
			info := beat.Info{
				IndexPrefix: "test",
				Version:     test.version,
			}

			im, err := idxmgmt.DefaultSupport(nil, info, c.NewConfig())
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
				clientSettings{
					observer:         outputs.NewNilObserver(),
					indexSelector:    index,
					pipelineSelector: pipeline,
				},
				nil,
			)
			assert.NoError(t, err)

			encoded, bulkItems := client.bulkEncodePublishRequest(*libversion.MustNew(test.version), events)
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
	cases := []mapstr.M{
		{"_id": "111", "op_type": e.OpTypeIndex, "message": "test 1", "bulkIndex": 0},
		{"_id": "112", "message": "test 2", "bulkIndex": 2},
		{"_id": "", "op_type": e.OpTypeDelete, "message": "test 6", "bulkIndex": -1}, // this won't get encoded due to missing _id
		{"_id": "", "message": "test 3", "bulkIndex": 4},
		{"_id": "114", "op_type": e.OpTypeDelete, "message": "test 4", "bulkIndex": 6},
		{"_id": "115", "op_type": e.OpTypeIndex, "message": "test 5", "bulkIndex": 7},
	}

	cfg := c.MustNewConfigFrom(mapstr.M{})
	info := beat.Info{
		IndexPrefix: "test",
		Version:     version.GetDefaultVersion(),
	}

	im, err := idxmgmt.DefaultSupport(nil, info, c.NewConfig())
	require.NoError(t, err)

	index, pipeline, err := buildSelectors(im, info, cfg)
	require.NoError(t, err)

	events := make([]publisher.Event, len(cases))
	for i, fields := range cases {
		meta := mapstr.M{
			"_id": fields["_id"],
		}
		if opType, exists := fields["op_type"]; exists {
			meta[e.FieldMetaOpType] = opType
		}

		events[i] = publisher.Event{
			Content: beat.Event{
				Meta: meta,
				Fields: mapstr.M{
					"message": fields["message"],
				},
			},
		}
	}

	client, _ := NewClient(
		clientSettings{
			observer:         outputs.NewNilObserver(),
			indexSelector:    index,
			pipelineSelector: pipeline,
		},
		nil,
	)

	encoded, bulkItems := client.bulkEncodePublishRequest(*libversion.MustNew(version.GetDefaultVersion()), events)
	require.Equal(t, len(events)-1, len(encoded), "all events should have been encoded")
	require.Equal(t, 9, len(bulkItems), "incomplete bulk")

	for i := 0; i < len(cases); i++ {
		bulkEventIndex, _ := cases[i]["bulkIndex"].(int)
		if bulkEventIndex == -1 {
			continue
		}
		caseOpType := cases[i]["op_type"]
		caseMessage := cases[i]["message"].(string)
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

	client, err := NewClient(clientSettings{
		observer: outputs.NewNilObserver(),
		connection: eslegclient.ConnectionSettings{
			URL:    ts.URL,
			APIKey: "hyokHG4BfWk5viKZ172X:o45JUkyuS--yiSAuuxl8Uw",
		},
	}, nil)
	assert.NoError(t, err)

	// This connection will fail since the server doesn't return a valid
	// response. This is fine since we're just testing the headers in the
	// original client request.
	//nolint:errcheck // connection doesn't need to succeed
	client.Connect()
	assert.Equal(t, "ApiKey aHlva0hHNEJmV2s1dmlLWjE3Mlg6bzQ1SlVreXVTLS15aVNBdXV4bDhVdw==", headers.Get("Authorization"))
}

func TestPublishEventsWithBulkFiltering(t *testing.T) {
	makePublishTestClient := func(t *testing.T, url string, configParams map[string]string) *Client {
		client, err := NewClient(
			clientSettings{
				observer: outputs.NewNilObserver(),
				connection: eslegclient.ConnectionSettings{
					URL:        url,
					Parameters: configParams,
				},
				indexSelector: testIndexSelector{},
			},
			nil,
		)
		require.NoError(t, err)
		return client
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	event1 := publisher.Event{Content: beat.Event{Fields: mapstr.M{"field": 1}}}

	t.Run("Single event with response filtering", func(t *testing.T) {
		var expectedFilteringParams = map[string]string{
			"filter_path": "errors,items.*.error,items.*.status",
		}
		var recParams url.Values

		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if strings.ContainsAny("_bulk", r.URL.Path) {
				recParams = r.URL.Query()
				response := []byte(`{"took":85,"errors":false,"items":[{"index":{"status":200}}]}`)
				_, _ = w.Write(response)
			}
			if strings.Contains("/", r.URL.Path) {
				response := []byte(`{}`)
				_, _ = w.Write(response)
			}
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL, nil)

		// Try publishing a batch that can be split
		events := []publisher.Event{event1}
		evt, err := client.publishEvents(ctx, events)
		require.NoError(t, err)
		require.Equal(t, len(recParams), len(expectedFilteringParams))
		require.Nil(t, evt)
	})
	t.Run("Single event with response filtering and preconfigured client params", func(t *testing.T) {
		var configParams = map[string]string{
			"hardcoded": "yes",
		}
		var expectedFilteringParams = map[string]string{
			"filter_path": "errors,items.*.error,items.*.status",
		}
		var recParams url.Values

		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if strings.ContainsAny("_bulk", r.URL.Path) {
				recParams = r.URL.Query()
				response := []byte(`{"took":85,"errors":false,"items":[{"index":{"status":200}}]}`)
				_, _ = w.Write(response)
			}
			if strings.Contains("/", r.URL.Path) {
				response := []byte(`{}`)
				_, _ = w.Write(response)
			}
		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL, configParams)

		// Try publishing a batch that can be split
		events := []publisher.Event{event1}
		evt, err := client.publishEvents(ctx, events)
		require.NoError(t, err)
		require.Equal(t, len(recParams), len(expectedFilteringParams)+len(configParams))
		require.Nil(t, evt)
	})
	t.Run("Single event without response filtering", func(t *testing.T) {
		var recParams url.Values

		esMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.ContainsAny("_bulk", r.URL.Path) {
				recParams = r.URL.Query()
				response := []byte(`{
					"took":85,
					"errors":false,
					"items":[
						{
							"index":{
								"_index":"test",
								"_id":"1",
								"_version":1,
								"result":"created",
								"_shards":{"total":2,"successful":1,"failed":0},
								"_seq_no":0,
								"_primary_term":1,
								"status":201
							}
						}
					]}`)
				_, _ = w.Write(response)
			}
			if strings.Contains("/", r.URL.Path) {
				response := []byte(`{}`)
				_, _ = w.Write(response)
			}
			w.WriteHeader(http.StatusOK)

		}))
		defer esMock.Close()
		client := makePublishTestClient(t, esMock.URL, nil)

		// Try publishing a batch that can be split
		events := []publisher.Event{event1}
		_, err := client.publishEvents(ctx, events)
		require.NoError(t, err)
		require.Equal(t, len(recParams), 1)
	})
}
