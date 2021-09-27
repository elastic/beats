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

package mage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// ExportDashboard exports a dashboard from Kibana and writes it into the given module.
func ExportDashboard() error {
	module := EnvOr("MODULE", "")
	if module == "" {
		return fmt.Errorf("MODULE must be specified")
	}

	id := EnvOr("ID", "")
	if id == "" {
		return fmt.Errorf("Dashboard ID must be specified")
	}

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	dashboardCmd := sh.RunCmd("go", "run", filepath.Join(beatsDir, "dev-tools/cmd/dashboards/export_dashboards.go"))

	folder := CWD("module", module)

	args := []string{
		"-folder", folder,
		"-dashboard", id,
	}
	if kibanaURL := EnvOr("KIBANA_URL", ""); kibanaURL != "" {
		args = append(args, "-kibana", kibanaURL)
	}

	return dashboardCmd(args...)
}

// ImportDashboards imports dashboards to Kibana using the Beat setup command.
//
// Depends on: build, dashboard
//
// Optional environment variables:
// - KIBANA_URL: URL of Kibana
// - KIBANA_ALWAYS: Connect to Kibana without checking ES version. Default true.
// - ES_URL: URL of Elasticsearch (only used with KIBANA_ALWAYS=false).
func ImportDashboards(buildDep, dashboardDep interface{}) error {
	mg.Deps(buildDep, dashboardDep)

	setupDashboards := sh.RunCmd(CWD(BeatName+binaryExtension(GOOS)),
		"setup", "--dashboards",
		"-E", "setup.dashboards.directory="+kibanaBuildDir)

	kibanaAlways := true
	if b, err := strconv.ParseBool(os.Getenv("KIBANA_ALWAYS")); err == nil {
		kibanaAlways = b
	}

	var args []string
	if kibanaURL := EnvOr("KIBANA_URL", ""); kibanaURL != "" {
		args = append(args, "-E", "setup.kibana.host="+kibanaURL)
	}
	if esURL := EnvOr("ES_URL", ""); !kibanaAlways && esURL != "" {
		args = append(args, "-E", "setup.elasticsearch.host="+esURL)
	}
	args = append(args, "-E", "setup.dashboards.always_kibana="+strconv.FormatBool(kibanaAlways))

	return setupDashboards(args...)
}
