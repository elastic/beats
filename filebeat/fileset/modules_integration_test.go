// +build integration

package fileset

import (
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/stretchr/testify/assert"
)

func TestLoadPipeline(t *testing.T) {
	client := elasticsearch.GetTestingElasticsearch()
	client.Request("DELETE", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)

	content := map[string]interface{}{
		"description": "describe pipeline",
		"processors": []map[string]interface{}{
			{
				"set": map[string]interface{}{
					"field": "foo",
					"value": "bar",
				},
			},
		},
	}

	err := loadPipeline(client, "my-pipeline-id", content)
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)
	assert.Equal(t, 200, status)
}

func TestSetupNginx(t *testing.T) {
	client := elasticsearch.GetTestingElasticsearch()
	client.Request("DELETE", "/_ingest/pipeline/nginx-access-with_plugins", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/nginx-error-pipeline", "", nil, nil)

	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)

	configs := []ModuleConfig{
		ModuleConfig{Module: "nginx"},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil)
	assert.NoError(t, err)

	err = reg.Setup(client)
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/_ingest/pipeline/nginx-access-with_plugins", "", nil, nil)
	assert.Equal(t, 200, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/nginx-error-pipeline", "", nil, nil)
	assert.Equal(t, 200, status)
}
