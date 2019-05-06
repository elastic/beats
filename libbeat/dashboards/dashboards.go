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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// ImportDashboards tries to import the kibana dashboards.
func ImportDashboards(
	ctx context.Context,
	beatInfo beat.Info, homePath string,
	kibanaConfig, dashboardsConfig *common.Config,
	msgOutputter MessageOutputter,
	pattern common.MapStr,
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

	if !kibanaConfig.Enabled() {
		return errors.New("kibana configuration missing for loading dashboards.")
	}

	return setupAndImportDashboardsViaKibana(ctx, beatInfo.Hostname, kibanaConfig, &dashConfig, msgOutputter, pattern)
}

func setupAndImportDashboardsViaKibana(ctx context.Context, hostname string, kibanaConfig *common.Config,
	dashboardsConfig *Config, msgOutputter MessageOutputter, fields common.MapStr) error {

	kibanaLoader, err := NewKibanaLoader(ctx, kibanaConfig, dashboardsConfig, hostname, msgOutputter)
	if err != nil {
		return fmt.Errorf("fail to create the Kibana loader: %v", err)
	}

	defer kibanaLoader.Close()

	kibanaLoader.statusMsg("Kibana URL %v", kibanaLoader.client.Connection.URL)

	return ImportDashboardsViaKibana(kibanaLoader, fields)
}

// ImportDashboardsViaKibana imports Dashboards to Kibana
func ImportDashboardsViaKibana(kibanaLoader *KibanaLoader, fields common.MapStr) error {
	version := kibanaLoader.version
	if !version.IsValid() {
		return errors.New("No valid kibana version available")
	}

	if !isKibanaAPIavailable(kibanaLoader.version) {
		return fmt.Errorf("Kibana API is not available in Kibana version %s", kibanaLoader.version.String())
	}

	importer, err := NewImporter(version, kibanaLoader.config, *kibanaLoader, fields)
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
