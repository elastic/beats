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
	"log"
	"path/filepath"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/elastic/beats/libbeat/kibana"

	errw "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// ImportDashboards tries to import the kibana dashboards.
func ImportDashboards(
	ctx context.Context,
	beatInfo beat.Info, homePath string,
	kibanaConfig, dashboardsConfig *common.Config,
	msgOutputter MessageOutputter, migration bool, fields []byte,
) error {
	if dashboardsConfig == nil || !dashboardsConfig.Enabled() {
		return nil
	}

	// unpack dashboard config
	dashConfig := defaultConfig
	dashConfig.Beat = beatInfo.Beat
	dashConfig.Dir = filepath.Join(homePath, defaultDirectory)
	err := dashboardsConfig.Unpack(&dashConfig)
	if err != nil {
		return err
	}

	// init kibana config object
	if kibanaConfig == nil {
		kibanaConfig = common.NewConfig()
	}

	if !kibanaConfig.Enabled() {
		return errors.New("kibana configuration missing for loading dashboards.")
	}

	// Generate index pattern
	version, _ := common.NewVersion("7.0.0") // TODO: dynamic version
	//indexP, _ := esConfig.String("index", 0)
	// TODO: What should we do about the index pattern. Kind of strange that ES configs are needed here.
	indexPattern, err := kibana.NewGenerator(beatInfo.Beat+"-*", beatInfo.Beat, fields, dashConfig.Dir, beatInfo.Version, *version, migration)
	if err != nil {
		log.Fatal(err)
	}

	pattern, err := indexPattern.Generate()
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	return setupAndImportDashboardsViaKibana(ctx, beatInfo.Hostname, kibanaConfig, &dashConfig, msgOutputter, pattern)

}

func setupAndImportDashboardsViaKibana(ctx context.Context, hostname string, kibanaConfig *common.Config,
	dashboardsConfig *Config, msgOutputter MessageOutputter, fields common.MapStr) error {

	kibanaLoader, err := NewKibanaLoader(ctx, kibanaConfig, dashboardsConfig, hostname, msgOutputter, fields)
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

	if !isKibanaAPIavailable(kibanaLoader.version) {
		return fmt.Errorf("Kibana API is not available in Kibana version %s", kibanaLoader.version.String())
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

func isKibanaAPIavailable(version common.Version) bool {
	return (version.Major == 5 && version.Minor >= 6) || version.Major >= 6
}
