// +build integration

package elasticsearch

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/internal"
	"github.com/elastic/beats/libbeat/outputs/outest"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

func TestClientConnect(t *testing.T) {
	client := getTestingElasticsearch(t)
	err := client.Connect()
	assert.NoError(t, err)
}

func TestClientPublishEvent(t *testing.T) {
	index := "beat-int-pub-single-event"
	output, client := connectTestEs(t, map[string]interface{}{
		"index": index,
	})

	// drop old index preparing test
	client.Delete(index, "", "", nil)

	batch := outest.NewBatch(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":    "libbeat",
			"message": "Test message from libbeat",
		},
	})

	err := output.Publish(batch)
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

	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

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

	publish := func(event beat.Event) {
		err := output.Publish(outest.NewBatch(event))
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

	publish(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":      "libbeat",
			"message":   "Test message 1",
			"pipeline":  pipeline,
			"testfield": 0,
		}})
	publish(beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"type":      "libbeat",
			"message":   "Test message 2",
			"testfield": 0,
		}})

	_, _, err = client.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, getCount("testfield:1")) // with pipeline 1
	assert.Equal(t, 1, getCount("testfield:0")) // no pipeline
}

func TestClientBulkPublishEventsWithPipeline(t *testing.T) {
	type obj map[string]interface{}

	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

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

	publish := func(events ...beat.Event) {
		err := output.Publish(outest.NewBatch(events...))
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
		beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":      "libbeat",
				"message":   "Test message 1",
				"pipeline":  pipeline,
				"testfield": 0,
			}},
		beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":      "libbeat",
				"message":   "Test message 2",
				"testfield": 0,
			}},
	)

	_, _, err = client.Refresh(index)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, getCount("testfield:1")) // with pipeline 1
	assert.Equal(t, 1, getCount("testfield:0")) // no pipeline
}

func connectTestEs(t *testing.T, cfg interface{}) (outputs.Client, *Client) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"hosts":            internal.GetEsHost(),
		"username":         internal.GetUser(),
		"password":         internal.GetPass(),
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

	output, err := makeES(beat.Info{Beat: "libbeat"}, outputs.NewNilObserver(), config)
	if err != nil {
		t.Fatal(err)
	}

	type clientWrap interface {
		outputs.NetworkClient
		Client() outputs.NetworkClient
	}
	client := randomClient(output).(clientWrap).Client().(*Client)

	// Load version number
	client.Connect()

	return client, client
}

// getTestingElasticsearch creates a test client.
func getTestingElasticsearch(t internal.TestLogger) *Client {
	client, err := NewClient(ClientSettings{
		URL:              internal.GetURL(),
		Index:            outil.MakeSelector(),
		Username:         internal.GetUser(),
		Password:         internal.GetUser(),
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)
	internal.InitClient(t, client, err)
	return client
}

func randomClient(grp outputs.Group) outputs.NetworkClient {
	L := len(grp.Clients)
	if L == 0 {
		panic("no elasticsearch client")
	}

	client := grp.Clients[rand.Intn(L)]
	return client.(outputs.NetworkClient)
}
