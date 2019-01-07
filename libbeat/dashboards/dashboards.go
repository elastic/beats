// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dashboards

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	errw "github.com/pkg/errors"

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

		if esLoader.version.Major < 6 {
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
	version := kibanaLoader.version
	if !version.IsValid() {
		return errors.New("No valid kibana version available")
	}

	if !isKibanaAPIavailable(version) {
		return fmt.Errorf("Kibana API is not available in Kibana version %s", version.String())
	}

	importer, err := NewImporter(version, kibanaLoader.config, kibanaLoader)
	if err != nil {
		return fmt.Errorf("fail to create a Kibana importer for loading the dashboards: %v", err)
	}

	if err := importer.Import(); err != nil {
		return errw.Wrap(err, "fail to import the dashboards in Kibana")
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

func isKibanaAPIavailable(version common.Version) bool {
	return (version.Major == 5 && version.Minor >= 6) || version.Major >= 6
}
