// +build integration

package elasticsearch

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"

	"github.com/stretchr/testify/assert"
)

func TestClientConnect(t *testing.T) {

	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.NoError(t, err)
}

func TestClientPublishEvent(t *testing.T) {
	index := "beat-int-pub-single-event"
	output, client := connectTestEs(t, map[string]interface{}{
		"index": index,
	})

	// drop old index preparing test
	client.Delete(index, "", "", nil)

	event := outputs.Data{Event: common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "libbeat",
		"message":    "Test message from libbeat",
	}}
	err := output.PublishEvent(nil, outputs.Options{Guaranteed: true}, event)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = client.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	_, resp, err := client.CountSearchURI(index, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, resp.Count)
}

func TestClientPublishEventWithPipeline(t *testing.T) {
	type obj map[string]interface{}

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := "beat-int-pub-single-with-pipeline"
	pipeline := "beat-int-pub-single-pipeline"

	output, client := connectTestEs(t, obj{
		"index":    index,
		"pipeline": "%{[pipeline]}",
	})
	client.Delete(index, "", "", nil)

	// Check version
	if strings.HasPrefix(client.Connection.version, "2.") {
		t.Skip("Skipping tests as pipeline not available in 2.x releases")
	}

	publish := func(event common.MapStr) {
		opts := outputs.Options{Guaranteed: true}
		err := output.PublishEvent(nil, opts, outputs.Data{Event: event})
		if err != nil {
			t.Fatal(err)
		}
	}

	getCount := func(query string) int {
		_, resp, err := client.CountSearchURI(index, "", map[string]string{
			"q": query,
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Count
	}

	pipelineBody := obj{
		"description": "Test pipeline",
		"processors": []obj{
			{
				"set": obj{
					"field": "testfield",
					"value": 1,
				},
			},
		},
	}

	client.DeletePipeline(pipeline, nil)
	_, resp, err := client.CreatePipeline(pipeline, nil, pipelineBody)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Acknowledged {
		t.Fatalf("Test pipeline %v not created", pipeline)
	}

	publish(common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "libbeat",
		"message":    "Test message 1",
		"pipeline":   pipeline,
		"testfield":  0,
	})
	publish(common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "libbeat",
		"message":    "Test message 2",
		"testfield":  0,
	})

	_, _, err = client.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, getCount("testfield:1")) // with pipeline 1
	assert.Equal(t, 1, getCount("testfield:0")) // no pipeline
}

func TestClientBulkPublishEventsWithPipeline(t *testing.T) {
	type obj map[string]interface{}

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := "beat-int-pub-bulk-with-pipeline"
	pipeline := "beat-int-pub-bulk-pipeline"

	output, client := connectTestEs(t, obj{
		"index":    index,
		"pipeline": "%{[pipeline]}",
	})
	client.Delete(index, "", "", nil)

	if strings.HasPrefix(client.Connection.version, "2.") {
		t.Skip("Skipping tests as pipeline not available in 2.x releases")
	}

	publish := func(events ...outputs.Data) {
		opts := outputs.Options{Guaranteed: true}
		err := output.BulkPublish(nil, opts, events)
		if err != nil {
			t.Fatal(err)
		}
	}

	getCount := func(query string) int {
		_, resp, err := client.CountSearchURI(index, "", map[string]string{
			"q": query,
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Count
	}

	pipelineBody := obj{
		"description": "Test pipeline",
		"processors": []obj{
			{
				"set": obj{
					"field": "testfield",
					"value": 1,
				},
			},
		},
	}

	client.DeletePipeline(pipeline, nil)
	_, resp, err := client.CreatePipeline(pipeline, nil, pipelineBody)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Acknowledged {
		t.Fatalf("Test pipeline %v not created", pipeline)
	}

	publish(
		outputs.Data{Event: common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       "libbeat",
			"message":    "Test message 1",
			"pipeline":   pipeline,
			"testfield":  0,
		}},
		outputs.Data{Event: common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       "libbeat",
			"message":    "Test message 2",
			"testfield":  0,
		}},
	)

	_, _, err = client.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, getCount("testfield:1")) // with pipeline 1
	assert.Equal(t, 1, getCount("testfield:0")) // no pipeline
}

func connectTestEs(t *testing.T, cfg interface{}) (outputs.BulkOutputer, *Client) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"hosts":            GetEsHost(),
		"username":         os.Getenv("ES_USER"),
		"password":         os.Getenv("ES_PASS"),
		"template.enabled": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	tmp, err := common.NewConfigFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = config.Merge(tmp)
	if err != nil {
		t.Fatal(err)
	}

	output, err := New(common.BeatInfo{Beat: "libbeat"}, config)
	if err != nil {
		t.Fatal(err)
	}

	es := output.(*elasticsearchOutput)
	client := es.randomClient()
	// Load version number
	client.Connect(3 * time.Second)

	return es, client
}
