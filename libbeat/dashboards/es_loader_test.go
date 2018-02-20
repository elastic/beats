// +build integration

package dashboards

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/estest"
)

func TestImporter(t *testing.T) {
	logp.TestingSetup()

	dashboardsConfig := Config{
		KibanaIndex: ".kibana-test",
		File:        "testdata/testbeat-dashboards.zip",
		Beat:        "testbeat",
	}

	client := estest.GetTestingElasticsearch(t)
	if strings.HasPrefix(client.Connection.GetVersion(), "6.") ||
		strings.HasPrefix(client.Connection.GetVersion(), "7.") {
		t.Skip("Skipping tests for Elasticsearch 6.x releases")
	}

	loader := ElasticsearchLoader{
		client: client,
		config: &dashboardsConfig,
	}

	err := loader.CreateKibanaIndex()

	assert.NoError(t, err)

	version, _ := common.NewVersion("5.0.0")

	imp, err := NewImporter(*version, &dashboardsConfig, loader)
	assert.NoError(t, err)

	err = imp.Import()
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/.kibana-test/dashboard/1e4389f0-e871-11e6-911d-3f8ed6f72700", "", nil, nil)
	assert.Equal(t, 200, status)
}

func TestImporterEmptyBeat(t *testing.T) {
	logp.TestingSetup()

	dashboardsConfig := Config{
		KibanaIndex: ".kibana-test-nobeat",
		File:        "testdata/testbeat-dashboards.zip",
		Beat:        "",
	}

	client := estest.GetTestingElasticsearch(t)
	if strings.HasPrefix(client.Connection.GetVersion(), "6.") ||
		strings.HasPrefix(client.Connection.GetVersion(), "7.") {
		t.Skip("Skipping tests for Elasticsearch 6.x releases")
	}

	loader := ElasticsearchLoader{
		client: client,
		config: &dashboardsConfig,
	}

	version, _ := common.NewVersion("5.0.0")

	imp, err := NewImporter(*version, &dashboardsConfig, loader)
	assert.NoError(t, err)

	err = imp.Import()
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/.kibana-test-nobeat/dashboard/1e4389f0-e871-11e6-911d-3f8ed6f72700", "", nil, nil)
	assert.Equal(t, 200, status)
}
