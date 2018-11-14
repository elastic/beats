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

package generator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/paths"
)

const (
	dashboardEntry = `- id: %s
  file: %s
`
)

func AddModuleDashboard(beatName, module, kibanaVersion, dashboardID string, dashboard common.MapStr, suffix string) error {
	version, err := common.NewVersion(kibanaVersion)
	if err != nil {
		return err
	}
	if version.Major < 6 {
		return fmt.Errorf("saving exported dashboards is not available for Kibana version '%s'", version.String())
	}

	modulePath := filepath.Join(paths.Resolve(paths.Home, "module"), module)
	stat, err := os.Stat(modulePath)
	if err != nil || !stat.IsDir() {
		return fmt.Errorf("no such module: %s\n", modulePath)
	}

	dashboardFile := strings.Title(beatName) + "-" + module + "-" + suffix + ".json"

	err = saveDashboardToFile(version, dashboard, dashboardFile, modulePath)
	if err != nil {
		return fmt.Errorf("cannot save dashboard to file: %+v", err)
	}

	return addDashboardToModuleYML(dashboardID, dashboardFile, modulePath)
}

func saveDashboardToFile(version *common.Version, dashboard common.MapStr, dashboardFile, modulePath string) error {
	dashboardsPath := "_meta/kibana/" + strconv.Itoa(version.Major) + "/dashboard"
	err := CreateDirectories(modulePath, dashboardsPath)
	if err != nil {
		return err
	}

	dashboardPath := filepath.Join(modulePath, dashboardsPath, dashboardFile)
	bytes, err := json.Marshal(dashboard)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(dashboardPath, bytes, 0644)
}

func addDashboardToModuleYML(dashboardID, dashboardFile, modulePath string) error {
	content := fmt.Sprintf(dashboardEntry, dashboardID, dashboardFile)

	f, err := os.OpenFile(filepath.Join(modulePath, "module.yml"), os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		_, err = f.Write([]byte(content))
	}

	return err
}
