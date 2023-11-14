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
	"errors"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

var (
	buildDep             interface{}
	collectDashboardsDep interface{}
)

// RegisterImportDeps registers dependencies of the Import target.
func RegisterImportDeps(build, collectDashboards interface{}) {
	buildDep = build
	collectDashboardsDep = collectDashboards
}

// Dashboards target namespace.
type Dashboards mg.Namespace

// Import imports dashboards to Kibana using the Beat setup command.
//
// Depends on: build, dashboard
//
// Optional environment variables:
// - KIBANA_URL: URL of Kibana
// - KIBANA_INSECURE: Disable TLS verification.
// - KIBANA_ALWAYS: Connect to Kibana without checking ES version. Default true.
// - ES_URL: URL of Elasticsearch (only used with KIBANA_ALWAYS=false).
func (Dashboards) Import() error {
	if buildDep == nil || collectDashboardsDep == nil {
		return errors.New("dashboard.RegisterImportDeps() must be called")
	}
	return devtools.ImportDashboards(buildDep, collectDashboardsDep)
}

// Export exports a dashboard from Kibana and writes it into the correct
// directory.
//
// Required environment variables:
// - KIBANA_URL: URL of Kibana
// - KIBANA_INSECURE: Disable TLS verification.
// - MODULE:     Name of the module
// - ID:         Dashboard ID
func (Dashboards) Export() error {
	return devtools.ExportDashboard()
}
