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
	"github.com/magefile/mage/sh"
)

// DefaultCleanPaths specifies a list of files or paths to recursively delete.
// The values may contain variables and will be expanded at the time of use.
var DefaultCleanPaths = []string{
	"build",
	"docker-compose.yml.lock",
	"{{.BeatName}}",
	"{{.BeatName}}.exe",
	"{{.BeatName}}.test",
	"{{.BeatName}}.test.exe",
	"fields.yml",
	"_meta/fields.generated.yml",
	"_meta/kibana.generated",
	"_meta/kibana/6/index-pattern/{{.BeatName}}.json",
	"_meta/kibana/7/index-pattern/{{.BeatName}}.json",
}

// Clean clean generated build artifacts.
func Clean(pathLists ...[]string) error {
	if len(pathLists) == 0 {
		pathLists = [][]string{DefaultCleanPaths}
	}
	for _, paths := range pathLists {
		for _, f := range paths {
			f = MustExpand(f)
			if err := sh.Rm(f); err != nil {
				return err
			}
		}
	}
	return nil
}
