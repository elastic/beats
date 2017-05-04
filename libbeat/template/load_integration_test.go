// +build integration

package template

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/version"

	"github.com/stretchr/testify/assert"
)

func TestCheckTemplate(t *testing.T) {

	client := elasticsearch.GetTestingElasticsearch()

	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	loader := &Loader{
		client: client,
	}

	// Check for non existent template
	assert.False(t, loader.CheckTemplate("libbeat-notexists"))
}

func TestLoadTemplate(t *testing.T) {

	// Setup ES
	client := elasticsearch.GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	// Load template
	absPath, err := filepath.Abs("../")
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	fieldsPath := absPath + "/fields.yml"
	index := "testbeat"

	tmpl, err := New(version.GetDefaultVersion(), client.GetVersion(), index)
	assert.NoError(t, err)
	content, err := tmpl.Load(fieldsPath)
	assert.NoError(t, err)

	loader := &Loader{
		client: client,
	}

	// Load template
	err = loader.LoadTemplate(tmpl.GetName(), content)
	assert.Nil(t, err)

	// Make sure template was loaded
	assert.True(t, loader.CheckTemplate(tmpl.GetName()))

	// Delete template again to clean up
	client.Request("DELETE", "/_template/"+tmpl.GetName(), "", nil, nil)

	// Make sure it was removed
	assert.False(t, loader.CheckTemplate(tmpl.GetName()))
}

func TestLoadInvalidTemplate(t *testing.T) {

	// Invalid Template
	template := map[string]interface{}{
		"json": "invalid",
	}

	// Setup ES
	client := elasticsearch.GetTestingElasticsearch()
	err := client.Connect(5 * time.Second)
	assert.Nil(t, err)

	templateName := "invalidtemplate"

	loader := &Loader{
		client: client,
	}

	// Try to load invalid template
	err = loader.LoadTemplate(templateName, template)
	assert.Error(t, err)

	// Make sure template was not loaded
	assert.False(t, loader.CheckTemplate(templateName))
}

// Tests loading the templates for each beat
func TestLoadBeatsTemplate(t *testing.T) {

	beats := []string{
		"libbeat",
	}

	for _, beat := range beats {
		// Load template
		absPath, err := filepath.Abs("../../" + beat)
		assert.NotNil(t, absPath)
		assert.Nil(t, err)

		// Setup ES
		client := elasticsearch.GetTestingElasticsearch()

		err = client.Connect(5 * time.Second)
		assert.Nil(t, err)

		fieldsPath := absPath + "/fields.yml"
		index := beat

		tmpl, err := New(version.GetDefaultVersion(), client.GetVersion(), index)
		assert.NoError(t, err)
		content, err := tmpl.Load(fieldsPath)
		assert.NoError(t, err)

		loader := &Loader{
			client: client,
		}

		// Load template
		err = loader.LoadTemplate(tmpl.GetName(), content)
		assert.Nil(t, err)

		// Make sure template was loaded
		assert.True(t, loader.CheckTemplate(tmpl.GetName()))

		// Delete template again to clean up
		client.Request("DELETE", "/_template/"+tmpl.GetName(), "", nil, nil)

		// Make sure it was removed
		assert.False(t, loader.CheckTemplate(tmpl.GetName()))
	}
}
