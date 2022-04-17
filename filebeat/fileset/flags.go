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

package fileset

import (
	"flag"
	"fmt"
	"strings"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// Modules related command line flags.
var (
	modulesFlag     = flag.String("modules", "", "List of enabled modules (comma separated)")
	moduleOverrides = common.SettingFlag(nil, "M", "Module configuration overwrite")
)

type ModuleOverrides map[string]map[string]*common.Config // module -> fileset -> Config

// Get returns an array of configuration overrides that should be merged in order.
func (mo *ModuleOverrides) Get(module, fileset string) []*common.Config {
	ret := []*common.Config{}

	moduleWildcard := (*mo)["*"]["*"]
	if moduleWildcard != nil {
		ret = append(ret, moduleWildcard)
	}

	filesetWildcard := (*mo)[module]["*"]
	if filesetWildcard != nil {
		ret = append(ret, filesetWildcard)
	}

	cfg := (*mo)[module][fileset]
	if cfg != nil {
		ret = append(ret, cfg)
	}

	return ret
}

func getModulesCLIConfig() ([]string, *ModuleOverrides, error) {
	modulesList := []string{}
	if modulesFlag != nil {
		modulesList = strings.Split(*modulesFlag, ",")
	}

	if moduleOverrides == nil {
		return modulesList, nil, nil
	}

	var overrides ModuleOverrides
	err := moduleOverrides.Unpack(&overrides)
	if err != nil {
		return []string{}, nil, fmt.Errorf("-M flags must be prefixed by the module and fileset: %v", err)
	}

	return modulesList, &overrides, nil
}
