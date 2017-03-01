package dashboards

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

// DashboardLoader is a subset of the Elasticsearch client API capable of
// loading the dashboards.
type DashboardLoader interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	CreateIndex(index string, body interface{}) (int, *elasticsearch.QueryResult, error)
}

func ImportDashboards(beatName, beatVersion string, esClient DashboardLoader, cfg *common.Config) error {
	if cfg == nil || !cfg.Enabled() {
		return nil
	}

	dashConfig := defaultDashboardsConfig
	dashConfig.Beat = beatName
	dashConfig.URL = fmt.Sprintf(defaultURLPattern, beatVersion)
	dashConfig.SnapshotURL = fmt.Sprintf(snapshotURLPattern, beatVersion)

	err := cfg.Unpack(&dashConfig)
	if err != nil {
		return err
	}

	importer, err := NewImporter(&dashConfig, esClient, nil)
	if err != nil {
		return nil
	}

	if err := importer.Import(); err != nil {
		return err
	}

	return nil
}
