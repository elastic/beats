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
	"path/filepath"

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
		return fmt.Errorf("Dashboad ID must be specified")
	}

	kibanaURL := EnvOr("KIBANA_URL", "")

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	// TODO: This is currently hardcoded for KB 7, we need to figure out what we do for KB 8 if applicable
	file := CWD("module", module, "_meta/kibana/7/dashboard", id+".json")

	dashboardCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, "dev-tools/cmd/dashboards/export_dashboards.go"),
		"-output", file, "-dashboard", id,
	)

	if kibanaURL != "" {
		return dashboardCmd("-kibana", kibanaURL)
	} else {
		return dashboardCmd()
	}
}
