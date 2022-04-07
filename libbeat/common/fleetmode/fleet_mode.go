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

package fleetmode

import (
	"flag"

	"github.com/elastic/beats/v8/libbeat/common"
)

// Enabled checks to see if filebeat/metricbeat is running under Agent
// The management setting is stored in the main Beat runtime object, but we can't see that from a module
// So instead we check the CLI flags, since Agent starts filebeat/metricbeat with "-E", "management.enabled=true"
func Enabled() bool {
	type management struct {
		Enabled bool `config:"management.enabled"`
	}
	var managementSettings management

	cfgFlag := flag.Lookup("E")
	if cfgFlag == nil {
		return false
	}

	cfgObject, _ := cfgFlag.Value.(*common.SettingsFlag)
	cliCfg := cfgObject.Config()

	err := cliCfg.Unpack(&managementSettings)
	if err != nil {
		return false
	}

	return managementSettings.Enabled
}
