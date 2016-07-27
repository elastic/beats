// +build integration

package elasticsearch

import (
	"testing"
	"time"

	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
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

	// Check for non existant template
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
	client.request("DELETE", "/_template/"+templateName, nil, nil)

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

		templatePath := absPath + "/" + beat + ".template.json"
		content, err := readTemplate(templatePath)
		assert.Nil(t, err)

		// Setup ES
		client := GetTestingElasticsearch()
		err = client.Connect(5 * time.Second)
		assert.Nil(t, err)

		templateName := beat

		// Load template
		err = client.LoadTemplate(templateName, content)
		assert.Nil(t, err)

		// Make sure template was loaded
		assert.True(t, client.CheckTemplate(templateName))

		// Delete template again to clean up
		client.request("DELETE", "/_template/"+templateName, nil, nil)

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
	client.request("DELETE", "/_template/libbeat", nil, nil)

	// Make sure template is not yet there
	assert.False(t, client.CheckTemplate("libbeat"))

	tPath, err := filepath.Abs("../../../packetbeat/packetbeat.template.json")
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
	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       "libbeat",
		"message":    "Test message from libbeat",
	}

	err = output.PublishEvent(nil, outputs.Options{Guaranteed: true}, event)
	if err != nil {
		t.Fatal(err)
	}

	// Guaranteed publish, so the template should be there

	assert.True(t, client.CheckTemplate("libbeat"))

}
