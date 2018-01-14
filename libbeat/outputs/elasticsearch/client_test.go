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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/outest"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/publisher"
)

func readStatusItem(in []byte) (int, string, error) {
	reader := newJSONReader(in)
	code, msg, err := itemStatus(reader)
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

	reader := newJSONReader(response)
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

	reader := newJSONReader(response)
	res, _ := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 1, len(res))
	if len(res) == 1 {
		assert.Equal(t, eventFail, res[0])
	}
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

	reader := newJSONReader(response)
	res, _ := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, events, res)
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

	reader := newJSONReader(response)
	res, _ := bulkCollectPublishFails(reader, events)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, events, res)
}

func TestGetIndexStandard(t *testing.T) {
	ts := time.Now().UTC()
	extension := fmt.Sprintf("%d.%02d.%02d", ts.Year(), ts.Month(), ts.Day())
	fields := common.MapStr{"field": 1}

	pattern := "beatname-%{+yyyy.MM.dd}"
	fmtstr := fmtstr.MustCompileEvent(pattern)
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	event := &beat.Event{Timestamp: ts, Fields: fields}
	index, _ := getIndex(event, indexSel)
	assert.Equal(t, index, "beatname-"+extension)
}

func TestGetIndexOverwrite(t *testing.T) {
	time := time.Now().UTC()
	extension := fmt.Sprintf("%d.%02d.%02d", time.Year(), time.Month(), time.Day())

	fields := common.MapStr{
		"@timestamp": common.Time(time),
		"field":      1,
		"beat": common.MapStr{
			"name": "testbeat",
		},
	}

	pattern := "beatname-%%{+yyyy.MM.dd}"
	fmtstr := fmtstr.MustCompileEvent(pattern)
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	event := &beat.Event{
		Timestamp: time,
		Meta: map[string]interface{}{
			"index": "dynamicindex",
		},
		Fields: fields}
	index, _ := getIndex(event, indexSel)
	expected := "dynamicindex-" + extension
	assert.Equal(t, expected, index)
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

	reader := newJSONReader(nil)
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

	reader := newJSONReader(nil)
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

	reader := newJSONReader(nil)
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
		fmt.Fprintln(w, "Hello, client")
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
