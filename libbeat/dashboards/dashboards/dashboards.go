package dashboards

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

func ImportDashboards(beatName, beatVersion string,
	kibanaConfig *common.Config, esConfig *common.Config,
	dashboardsConfig *common.Config) error {

	if dashboardsConfig == nil || !dashboardsConfig.Enabled() {
		return nil
	}

	dashConfig := defaultConfig
	dashConfig.Beat = beatName
	dashConfig.URL = fmt.Sprintf(defaultURLPattern, beatVersion)
	dashConfig.SnapshotURL = fmt.Sprintf(snapshotURLPattern, beatVersion)

	err := dashboardsConfig.Unpack(&dashConfig)
	if err != nil {
		return err
	}

	if esConfig != nil {
		status, err := ImportDashboardsViaElasticsearch(esConfig, &dashConfig, nil)
		if err != nil {
			return err
		}
		if status {
			// the dashboards were imported via Elasticsearch
			return nil
		}
	}

	err = ImportDashboardsViaKibana(kibanaConfig, &dashConfig, nil)
	if err != nil {
		return err
	}

	return nil
}

func ImportDashboardsViaKibana(config *common.Config, dashConfig *Config, msgOutputter MessageOutputter) error {
	if config == nil {
		config = common.NewConfig()
	}
	if !config.Enabled() {
		return nil
	}

	kibanaLoader, err := NewKibanaLoader(config, dashConfig, msgOutputter)
	if err != nil {
		return fmt.Errorf("fail to create the Kibana loader: %v", err)
	}

	defer kibanaLoader.Close()

	if !isKibanaAPIavailable(kibanaLoader.version) {
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

func ImportDashboardsViaElasticsearch(config *common.Config, dashConfig *Config, msgOutputter MessageOutputter) (bool, error) {
	esLoader, err := NewElasticsearchLoader(config, dashConfig, msgOutputter)
	if err != nil {
		return false, fmt.Errorf("fail to create the Elasticsearch loader: %v", err)
	}
	defer esLoader.Close()

	esLoader.statusMsg("Elasticsearch URL %v", esLoader.client.Connection.URL)

	majorVersion, _, err := getMajorAndMinorVersion(esLoader.version)
	if err != nil {
		return false, fmt.Errorf("wrong Elasticsearch version: %v", err)
	}

	if majorVersion >= 6 {
		esLoader.statusMsg("For Elasticsearch version >= 6.0.0, the Kibana dashboards need to be imported via the Kibana API.")
		return false, nil
	}

	if err := esLoader.CreateKibanaIndex(); err != nil {
		return false, fmt.Errorf("fail to create the kibana index: %v", err)
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

func getMajorAndMinorVersion(version string) (int, int, error) {
	fields := strings.Split(version, ".")
	if len(fields) != 3 {
		return 0, 0, fmt.Errorf("wrong version %s", version)
	}
	majorVersion := fields[0]
	minorVersion := fields[1]

	majorVersionInt, err := strconv.Atoi(majorVersion)
	if err != nil {
		return 0, 0, err
	}

	minorVersionInt, err := strconv.Atoi(minorVersion)
	if err != nil {
		return 0, 0, err
	}

	return majorVersionInt, minorVersionInt, nil
}

func isKibanaAPIavailable(version string) bool {

	majorVersion, minorVersion, err := getMajorAndMinorVersion(version)
	if err != nil {
		return false
	}

	if majorVersion == 5 && minorVersion >= 6 {
		return true
	}

	if majorVersion >= 6 {
		return true
	}

	return false
}
