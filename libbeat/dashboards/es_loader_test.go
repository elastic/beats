// +build integration

package dashboards

import (
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/stretchr/testify/assert"
)

func TestImporter(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	dashboardsConfig := DashboardsConfig{
		KibanaIndex: ".kibana-test",
		File:        "testdata/testbeat-dashboards.zip",
		Beat:        "testbeat",
	}

	client := elasticsearch.GetTestingElasticsearch(t)
	if strings.HasPrefix(client.Connection.GetVersion(), "6.") {
		t.Skip("Skipping tests for Elasticsearch 6.x releases")
	}

	loader := ElasticsearchLoader{
		client: client,
		config: &dashboardsConfig,
	}

	err := loader.createKibanaIndex()

	assert.NoError(t, err)

	imp, err := NewImporter("5.x", &dashboardsConfig, loader)

	assert.NoError(t, err)

	err = imp.Import()
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/.kibana-test/dashboard/1e4389f0-e871-11e6-911d-3f8ed6f72700", "", nil, nil)
	assert.Equal(t, 200, status)

}

func TestImporterEmptyBeat(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	dashboardsConfig := DashboardsConfig{
		KibanaIndex: ".kibana-test-nobeat",
		File:        "testdata/testbeat-dashboards.zip",
		Beat:        "",
	}

	client := elasticsearch.GetTestingElasticsearch(t)
	if strings.HasPrefix(client.Connection.GetVersion(), "6.") {
		t.Skip("Skipping tests for Elasticsearch 6.x releases")
	}

	loader := ElasticsearchLoader{
		client: client,
		config: &dashboardsConfig,
	}

	imp, err := NewImporter("5.x", &dashboardsConfig, loader)

	assert.NoError(t, err)

	err = imp.Import()
	assert.NoError(t, err)

	status, _, _ := client.Request("GET", "/.kibana-test-nobeat/dashboard/1e4389f0-e871-11e6-911d-3f8ed6f72700", "", nil, nil)
	assert.Equal(t, 200, status)

}
