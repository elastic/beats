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
	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// GenerateOSSMetricbeatModuleIncludeListGo generates include/list_{suffix}.go files containing
// a import statement for each module and dataset.
func GenerateOSSMetricbeatModuleIncludeListGo() error {
	// generate include/list_common.go
	err := devtools.GenerateIncludeListGo(
		devtools.IncludeListOptions{
			ImportDirs:       nil,
			ModuleDirs:       []string{"module"},
			ModulesToExclude: []string{"module/docker", "module/kubernetes"},
			Outfile:          "include/list_common.go",
			BuildTags:        "",
			Pkg:              "include",
			SkipInitModule:   true,
		})
	if err != nil {
		return err
	}
	// generate include/list_docker.go
	err = devtools.GenerateIncludeListGo(
		devtools.IncludeListOptions{
			ImportDirs:       nil,
			ModuleDirs:       []string{"module/docker", "module/kubernetes"},
			ModulesToExclude: nil,
			Outfile:          "include/list_docker.go",
			BuildTags:        "\n//go:build linux || darwin || windows\n",
			Pkg:              "include",
			SkipInitModule:   true,
		})
	if err != nil {
		return err
	}
	// generate include/list_init.go
	err = devtools.GenerateIncludeListGo(
		devtools.IncludeListOptions{
			ImportDirs:       []string{"autodiscover/**/*", "autodiscover/**/*/*", "processor/*"},
			ModuleDirs:       nil,
			ModulesToExclude: nil,
			Outfile:          "include/list_init.go",
			Pkg:              "include",
			SkipInitModule:   false,
		})
	if err != nil {
		return err
	}
	return nil
}
