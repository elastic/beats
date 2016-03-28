// +build integration

package elasticsearch

import (
	"testing"
	"time"

	"bytes"
	"io/ioutil"
	"path/filepath"

	"github.com/stretchr/testify/assert"
)

func TestClientConnect(t *testing.T) {

	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)

	assert.Nil(t, err)
	assert.True(t, client.IsConnected())
}

func TestCheckTemplate(t *testing.T) {

	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	// Check for non existant template
	assert.False(t, client.CheckTemplate("libbeat"))
}

func TestLoadTemplate(t *testing.T) {

	// Load template
	absPath, err := filepath.Abs("../../tests/files/")
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	templatePath := absPath + "/template.json"
	content, err := ioutil.ReadFile(templatePath)
	reader := bytes.NewReader(content)
	assert.Nil(t, err)

	// Setup ES
	client := GetTestingElasticsearch()
	err = client.Connect(5 * time.Second)
	assert.Nil(t, err)

	templateName := "testbeat"

	// Load template
	err = client.LoadTemplate(templateName, reader)
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
	reader := bytes.NewReader([]byte("{json:invalid}"))

	// Setup ES
	client := GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	templateName := "invalidtemplate"

	// Try to load invalid template
	err = client.LoadTemplate(templateName, reader)
	assert.Error(t, err)

	// Make sure template was not loaded
	assert.False(t, client.CheckTemplate(templateName))
}

// Tests loading the templates for each beat
func TestLoadBeatsTemplate(t *testing.T) {

	beats := []string{
		"topbeat",
		"filebeat",
		"packetbeat",
		"metricbeat",
		"winlogbeat",
	}

	for _, beat := range beats {
		// Load template
		absPath, err := filepath.Abs("../../../" + beat + "/etc/")
		assert.NotNil(t, absPath)
		assert.Nil(t, err)

		templatePath := absPath + "/" + beat + ".template.json"
		content, err := ioutil.ReadFile(templatePath)
		reader := bytes.NewReader(content)
		assert.Nil(t, err)

		// Setup ES
		client := GetTestingElasticsearch()
		err = client.Connect(5 * time.Second)
		assert.Nil(t, err)

		templateName := beat

		// Load template
		err = client.LoadTemplate(templateName, reader)
		assert.Nil(t, err)

		// Make sure template was loaded
		assert.True(t, client.CheckTemplate(templateName))

		// Delete template again to clean up
		client.request("DELETE", "/_template/"+templateName, nil, nil)

		// Make sure it was removed
		assert.False(t, client.CheckTemplate(templateName))
	}
}
