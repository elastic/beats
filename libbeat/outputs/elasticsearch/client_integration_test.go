// +build integration

package elasticsearch

import (
	"os"
	"strings"
	"testing"
	"time"

	"path/filepath"

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

func TestCheckTemplate(t *testing.T) {

	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	// Check for non existent template
	assert.False(t, client.CheckTemplate("libbeat-notexists"))
}

func TestLoadTemplate(t *testing.T) {

	// Load template
	absPath, err := filepath.Abs("../../tests/files/")
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	templatePath := absPath + "/template.json"
	content, err := readTemplate(templatePath)
	assert.Nil(t, err)

	// Setup ES
	client := GetTestingElasticsearch()
	err = client.Connect(5 * time.Second)
	assert.Nil(t, err)

	templateName := "testbeat"

	// Load template
	err = client.LoadTemplate(templateName, content)
	assert.Nil(t, err)

	// Make sure template was loaded
	assert.True(t, client.CheckTemplate(templateName))

	// Delete template again to clean up
	client.request("DELETE", "/_template/"+templateName, "", nil, nil)

	// Make sure it was removed
	assert.False(t, client.CheckTemplate(templateName))

}

func TestLoadInvalidTemplate(t *testing.T) {

	// Invalid Template
	template := map[string]interface{}{
		"json": "invalid",
	}

	// Setup ES
	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	templateName := "invalidtemplate"

	// Try to load invalid template
	err = client.LoadTemplate(templateName, template)
	assert.Error(t, err)

	// Make sure template was not loaded
	assert.False(t, client.CheckTemplate(templateName))
}

// Tests loading the templates for each beat
func TestLoadBeatsTemplate(t *testing.T) {

	beats := []string{
		"filebeat",
		"packetbeat",
		"metricbeat",
		"winlogbeat",
	}

	for _, beat := range beats {
		// Load template
		absPath, err := filepath.Abs("../../../" + beat)
		assert.NotNil(t, absPath)
		assert.Nil(t, err)

		// Setup ES
		client := GetTestingElasticsearch()

		templatePath := absPath + "/" + beat + ".template.json"

		if strings.HasPrefix(client.Connection.version, "2.") {
			templatePath = absPath + "/" + beat + ".template-es2x.json"
		}

		content, err := readTemplate(templatePath)
		assert.Nil(t, err)

		err = client.Connect(5 * time.Second)
		assert.Nil(t, err)

		templateName := beat

		// Load template
		err = client.LoadTemplate(templateName, content)
		assert.Nil(t, err)

		// Make sure template was loaded
		assert.True(t, client.CheckTemplate(templateName))

		// Delete template again to clean up
		client.request("DELETE", "/_template/"+templateName, "", nil, nil)

		// Make sure it was removed
		assert.False(t, client.CheckTemplate(templateName))
	}
}

// TestOutputLoadTemplate checks that the template is inserted before
// the first event is published.
func TestOutputLoadTemplate(t *testing.T) {

	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// delete template if it exists
	client.request("DELETE", "/_template/libbeat", "", nil, nil)

	// Make sure template is not yet there
	assert.False(t, client.CheckTemplate("libbeat"))

	templatePath := "../../../packetbeat/packetbeat.template.json"

	if strings.HasPrefix(client.Connection.version, "2.") {
		templatePath = "../../../packetbeat/packetbeat.template-es2x.json"
	}

	tPath, err := filepath.Abs(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	config := map[string]interface{}{
		"hosts": GetEsHost(),
		"template": map[string]interface{}{
			"name":                "libbeat",
			"path":                tPath,
			"versions.2x.enabled": false,
		},
	}

	cfg, err := common.NewConfigFrom(config)
	if err != nil {
		t.Fatal(err)
	}

	output, err := New("libbeat", cfg, 0)
	if err != nil {
		t.Fatal(err)
	}
	event := outputs.Data{Event: common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       "libbeat",
		"message":    "Test message from libbeat",
	}}

	err = output.PublishEvent(nil, outputs.Options{Guaranteed: true}, event)
	if err != nil {
		t.Fatal(err)
	}

	// Guaranteed publish, so the template should be there

	assert.True(t, client.CheckTemplate("libbeat"))

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

	output, err := New("libbeat", config, 0)
	if err != nil {
		t.Fatal(err)
	}

	es := output.(*elasticsearchOutput)
	client := es.randomClient()
	// Load version number
	client.Connect(3 * time.Second)

	return es, client
}
