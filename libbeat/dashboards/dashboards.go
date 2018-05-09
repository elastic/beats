package dashboards

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

type importMethod uint8

// check import route
const (
	importNone importMethod = iota
	importViaKibana
	importViaES
)

// ImportDashboards tries to import the kibana dashboards.
// If the Elastic Stack is at version 6.0+, the dashboards should be installed
// via the kibana dashboard loader plugin. For older versions of the Elastic Stack
// we write the dashboards directly into the .kibana index.
func ImportDashboards(
	ctx context.Context,
	beatName, hostname, homePath string,
	kibanaConfig, esConfig, dashboardsConfig *common.Config,
	msgOutputter MessageOutputter,
) error {
	if dashboardsConfig == nil || !dashboardsConfig.Enabled() {
		return nil
	}

	// unpack dashboard config
	dashConfig := defaultConfig
	dashConfig.Beat = beatName
	dashConfig.Dir = filepath.Join(homePath, defaultDirectory)
	err := dashboardsConfig.Unpack(&dashConfig)
	if err != nil {
		return err
	}

	// init kibana config object
	if kibanaConfig == nil {
		kibanaConfig = common.NewConfig()
	}

	if esConfig.Enabled() {
		username, _ := esConfig.String("username", -1)
		password, _ := esConfig.String("password", -1)

		if !kibanaConfig.HasField("username") && username != "" {
			kibanaConfig.SetString("username", -1, username)
		}
		if !kibanaConfig.HasField("password") && password != "" {
			kibanaConfig.SetString("password", -1, password)
		}
	}

	var esLoader *ElasticsearchLoader

	importVia := importNone
	useKibana := importViaKibana
	if !kibanaConfig.Enabled() {
		useKibana = importNone
	}

	requiresKibana := dashConfig.AlwaysKibana || !esConfig.Enabled()
	if requiresKibana {
		importVia = useKibana
	} else {
		// Check import route via elasticsearch version. If Elasticsearch major
		// version is >6, we assume Kibana also being at versions >6.0. In this
		// case dashboards will be imported using the new kibana dashboard loader
		// plugin.
		// XXX(urso): Why do we test the Elasticsearch version? If kibana is
		//            configured, why not test the kibana version and plugin
		//            availability first?
		esLoader, err = NewElasticsearchLoader(esConfig, &dashConfig, msgOutputter)
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
			importVia = importViaES
		} else {
			importVia = useKibana
		}
	}

	// Try to import dashboards.
	switch importVia {
	case importViaES:
		return ImportDashboardsViaElasticsearch(esLoader)
	case importViaKibana:
		return setupAndImportDashboardsViaKibana(ctx, hostname, kibanaConfig, &dashConfig, msgOutputter)
	default:
		return errors.New("Elasticsearch or Kibana configuration missing for loading dashboards.")
	}
}

func setupAndImportDashboardsViaKibana(ctx context.Context, hostname string, kibanaConfig *common.Config,
	dashboardsConfig *Config, msgOutputter MessageOutputter) error {

	kibanaLoader, err := NewKibanaLoader(ctx, kibanaConfig, dashboardsConfig, hostname, msgOutputter)
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
