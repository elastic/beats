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
	"path/filepath"

	"github.com/magefile/mage/sh"
)

// GenerateFieldsYAML generates a fields.yml file for a Beat. This will include
// the common fields specified by libbeat, the common fields for the Beat,
// and any additional fields.yml files you specify.
//
// fieldsFiles specifies additional directories to search recursively for files
// named fields.yml. The contents of each fields.yml will be included in the
// generated file.
func GenerateFieldsYAML(fieldsFiles ...string) error {
	const globalFieldsCmdPath = "libbeat/scripts/cmd/global_fields/main.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	globalFieldsCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, globalFieldsCmdPath),
		"-es_beats_path", beatsDir,
		"-beat_path", CWD(),
		"-out", "fields.yml",
	)

	return globalFieldsCmd(fieldsFiles...)
}
