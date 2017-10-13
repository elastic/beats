package dashboards

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func ImportDashboards(beatName, hostname, homePath string,
	kibanaConfig *common.Config, esConfig *common.Config,
	dashboardsConfig *common.Config, msgOutputter MessageOutputter) error {

	if dashboardsConfig == nil || !dashboardsConfig.Enabled() {
		return nil
	}

	dashConfig := defaultConfig
	dashConfig.Beat = beatName
	if dashConfig.Dir == "" {
		dashConfig.Dir = filepath.Join(homePath, defaultDirectory)
	}

	err := dashboardsConfig.Unpack(&dashConfig)
	if err != nil {
		return err
	}

	esLoader, err := NewElasticsearchLoader(esConfig, &dashConfig, msgOutputter)
	if err != nil {
		return fmt.Errorf("fail to create the Elasticsearch loader: %v", err)
	}
	defer esLoader.Close()

	esLoader.statusMsg("Elasticsearch URL %v", esLoader.client.Connection.URL)

	majorVersion, _, err := getMajorAndMinorVersion(esLoader.version)
	if err != nil {
		return fmt.Errorf("wrong Elasticsearch version: %v", err)
	}

	if majorVersion < 6 {
		return ImportDashboardsViaElasticsearch(esLoader)
	}

	logp.Info("For Elasticsearch version >= 6.0.0, the Kibana dashboards need to be imported via the Kibana API.")

	if kibanaConfig == nil {
		kibanaConfig = common.NewConfig()
	}

	// In Cloud, the Kibana URL is different than the Elasticsearch URL,
	// but the credentials are the same.
	// So, by default, use same credentials for connecting to Kibana as to Elasticsearch
	if !kibanaConfig.HasField("username") && len(esLoader.client.Username) > 0 {
		kibanaConfig.SetString("username", -1, esLoader.client.Username)
	}
	if !kibanaConfig.HasField("password") && len(esLoader.client.Password) > 0 {
		kibanaConfig.SetString("password", -1, esLoader.client.Password)
	}

	kibanaLoader, err := NewKibanaLoader(kibanaConfig, &dashConfig, hostname, msgOutputter)
	if err != nil {
		return fmt.Errorf("fail to create the Kibana loader: %v", err)
	}

	defer kibanaLoader.Close()

	kibanaLoader.statusMsg("Kibana URL %v", kibanaLoader.client.Connection.URL)

	return ImportDashboardsViaKibana(kibanaLoader)
}

func ImportDashboardsViaKibana(kibanaLoader *KibanaLoader) error {

	if !isKibanaAPIavailable(kibanaLoader.version) {
		return fmt.Errorf("Kibana API is not available in Kibana version %s", kibanaLoader.version)
	}

	version, err := common.NewVersion(kibanaLoader.version)
	if err != nil {
		return fmt.Errorf("Invalid Kibana version: %s", kibanaLoader.version)
	}

	importer, err := NewImporter(*version, kibanaLoader.config, kibanaLoader)
	if err != nil {
		return fmt.Errorf("fail to create a Kibana importer for loading the dashboards: %v", err)
	}

	if err := importer.Import(); err != nil {
		return fmt.Errorf("fail to import the dashboards in Kibana: %v", err)
	}

	return nil
}

func ImportDashboardsViaElasticsearch(esLoader *ElasticsearchLoader) error {

	if err := esLoader.CreateKibanaIndex(); err != nil {
		return fmt.Errorf("fail to create the kibana index: %v", err)
	}

	version, _ := common.NewVersion("5.0.0")

	importer, err := NewImporter(*version, esLoader.config, esLoader)
	if err != nil {
		return fmt.Errorf("fail to create an Elasticsearch importer for loading the dashboards: %v", err)
	}

	if err := importer.Import(); err != nil {
		return fmt.Errorf("fail to import the dashboards in Elasticsearch: %v", err)
	}

	return nil
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
