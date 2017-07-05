package dashboards

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func ImportDashboards(beatName, beatVersion string, kibanaConfig *common.Config, esConfig *common.Config,
	dashboardsConfig *common.Config) error {
	if dashboardsConfig == nil || !dashboardsConfig.Enabled() {
		return nil
	}

	dashConfig := defaultDashboardsConfig
	dashConfig.Beat = beatName
	dashConfig.URL = fmt.Sprintf(defaultURLPattern, beatVersion)
	dashConfig.SnapshotURL = fmt.Sprintf(snapshotURLPattern, beatVersion)

	err := dashboardsConfig.Unpack(&dashConfig)
	if err != nil {
		return err
	}

	if esConfig != nil {
		status, err := ImportDashboardsViaElasticsearch(esConfig, &dashConfig)
		if err != nil {
			return err
		}
		if status == true {
			// the dashboards were imported via Elasticsearch
			return nil
		}
	}

	err = ImportDashboardsViaKibana(kibanaConfig, &dashConfig)
	if err != nil {
		return err
	}

	return nil
}

func ImportDashboardsViaKibana(config *common.Config, dashConfig *DashboardsConfig) error {

	if config == nil {
		config = common.NewConfig()
	}
	if !config.Enabled() {
		return nil
	}

	kibanaLoader, err := NewKibanaLoader(config, dashConfig, nil)
	if err != nil {
		return fmt.Errorf("fail to create the Kibana loader: %v", err)
	}

	defer kibanaLoader.Close()

	version, err := getMajorVersion(kibanaLoader.version)
	if err != nil {
		return fmt.Errorf("wrong Kibana version: %v", err)
	}

	if version < 6 {
		return fmt.Errorf("Kibana API is not available in Kibana version %s", kibanaLoader.version)
	}

	importer, err := NewImporter("default", dashConfig, *kibanaLoader)
	if err != nil {
		return fmt.Errorf("fail to create a Kibana importer for loading the dashboards: %v", err)
	}

	if err := importer.Import(); err != nil {
		return fmt.Errorf("fail to import the dashboards in Kibana: %v", err)
	}

	return nil
}

func ImportDashboardsViaElasticsearch(config *common.Config, dashConfig *DashboardsConfig) (bool, error) {

	esLoader, err := NewElasticsearchLoader(config, dashConfig, nil)
	if err != nil {
		return false, fmt.Errorf("fail to create the Elasticsearch loader: %v", err)
	}

	defer esLoader.Close()

	logp.Debug("dashboards", "Elasticsearch URL %v", esLoader.client.Connection.URL)

	version, err := getMajorVersion(esLoader.version)
	if err != nil {
		return false, fmt.Errorf("wrong Elasticsearch version: %v", err)
	}

	if version >= 6 {
		logp.Info("For Elasticsearch version >= 6.0.0, the Kibana dashboards need to be imported via the Kibana API.")
		return false, nil
	}

	importer, err := NewImporter("5.x", dashConfig, *esLoader)
	if err != nil {
		return false, fmt.Errorf("fail to create an Elasticsearch importer for loading the dashboards: %v", err)
	}

	if err := importer.Import(); err != nil {
		return false, fmt.Errorf("fail to import the dashboards in Elasticsearch: %v", err)
	}

	return true, nil

}
func getMajorVersion(version string) (int, error) {

	fields := strings.Split(version, ".")
	if len(fields) != 3 {
		return 0, fmt.Errorf("wrong version %s", version)
	}
	majorVersion := fields[0]
	majorVersionInt, err := strconv.Atoi(majorVersion)
	if err != nil {
		return 0, err
	}

	return majorVersionInt, nil
}
