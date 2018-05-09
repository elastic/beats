// +build integration

package fileset

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/estest"
)

func TestLoadPipeline(t *testing.T) {
	client := estest.GetTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}

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

	err := loadPipeline(client, "my-pipeline-id", content, false)
	assert.NoError(t, err)

	status, _, err := client.Request("GET", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, status)

	// loading again shouldn't actually update the pipeline
	content["description"] = "describe pipeline 2"
	err = loadPipeline(client, "my-pipeline-id", content, false)
	assert.NoError(t, err)
	checkUploadedPipeline(t, client, "describe pipeline")

	// loading again updates the pipeline
	err = loadPipeline(client, "my-pipeline-id", content, true)
	assert.NoError(t, err)
	checkUploadedPipeline(t, client, "describe pipeline 2")
}

func checkUploadedPipeline(t *testing.T, client *elasticsearch.Client, expectedDescription string) {
	status, response, err := client.Request("GET", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, status)

	var res map[string]interface{}
	err = json.Unmarshal(response, &res)
	if assert.NoError(t, err) {
		assert.Equal(t, expectedDescription, res["my-pipeline-id"].(map[string]interface{})["description"], string(response))
	}
}

func TestSetupNginx(t *testing.T) {
	client := estest.GetTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}

	client.Request("DELETE", "/_ingest/pipeline/filebeat-5.2.0-nginx-access-default", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/filebeat-5.2.0-nginx-error-pipeline", "", nil, nil)

	modulesPath, err := filepath.Abs("../module")
	assert.NoError(t, err)

	configs := []*ModuleConfig{
		&ModuleConfig{Module: "nginx"},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, "5.2.0")
	if err != nil {
		t.Fatal(err)
	}

	err = reg.LoadPipelines(client, false)
	if err != nil {
		t.Fatal(err)
	}

	status, _, _ := client.Request("GET", "/_ingest/pipeline/filebeat-5.2.0-nginx-access-default", "", nil, nil)
	assert.Equal(t, 200, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/filebeat-5.2.0-nginx-error-pipeline", "", nil, nil)
	assert.Equal(t, 200, status)
}

func TestAvailableProcessors(t *testing.T) {
	client := estest.GetTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}
	// these exists on our integration test setup
	requiredProcessors := []ProcessorRequirement{
		{Name: "user_agent", Plugin: "ingest-user-agent"},
		{Name: "geoip", Plugin: "ingest-geoip"},
	}

	err := checkAvailableProcessors(client, requiredProcessors)
	assert.NoError(t, err)

	// these don't exists on our integration test setup
	requiredProcessors = []ProcessorRequirement{
		{Name: "test", Plugin: "ingest-test"},
		{Name: "hello", Plugin: "ingest-hello"},
	}

	err = checkAvailableProcessors(client, requiredProcessors)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "ingest-test")
	assert.Contains(t, err.Error(), "ingest-hello")
}

func hasIngest(client *elasticsearch.Client) bool {
	v := client.GetVersion()
	majorVersion := string(v[0])
	version, err := strconv.Atoi(majorVersion)
	if err != nil {
		return true
	}

	return version >= 5
}
