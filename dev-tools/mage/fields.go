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
// moduleDirs specifies additional directories to search for modules. The
// contents of each fields.yml will be included in the generated file.
func GenerateFieldsYAML(moduleDirs ...string) error {
	return generateFieldsYAML(OSSBeatDir(), moduleDirs...)
}

func generateFieldsYAML(baseDir string, moduleDirs ...string) error {
	const globalFieldsCmdPath = "libbeat/scripts/cmd/global_fields/main.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	globalFieldsCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, globalFieldsCmdPath),
		"-es_beats_path", beatsDir,
		"-beat_path", baseDir,
		"-out", "fields.yml",
	)

	return globalFieldsCmd(moduleDirs...)
}

// GenerateAllInOneFieldsGo generates an all-in-one fields.go file.
func GenerateAllInOneFieldsGo() error {
	return GenerateFieldsGo("fields.yml", "include/fields.go")
}

// GenerateFieldsGo generates a .go file containing the fields.yml data.
func GenerateFieldsGo(fieldsYML, out string) error {
	const assetCmdPath = "dev-tools/cmd/asset/asset.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	licenseType := BeatLicense
	if licenseType == "ASL 2.0" {
		licenseType = "ASL2"
	}

	assetCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, assetCmdPath),
		"-pkg", "include",
		"-in", fieldsYML,
		"-out", createDir(out),
		"-license", licenseType,
		BeatName,
	)

	return assetCmd()
}

// GenerateModuleFieldsGo generates a fields.go file containing a copy of the
// each module's field.yml data in a format that can be embedded in Beat's
// binary.
func GenerateModuleFieldsGo() error {
	const moduleFieldsCmdPath = "dev-tools/cmd/module_fields/module_fields.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	licenseType := BeatLicense
	if licenseType == "ASL 2.0" {
		licenseType = "ASL2"
	}

	moduleFieldsCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, moduleFieldsCmdPath),
		"-beat", BeatName,
		"-license", licenseType,
		filepath.Join(CWD(), "module"),
	)

	return moduleFieldsCmd()
}

// GenerateModuleIncludeListGo generates an include/list.go file containing
// a import statement for each module and metricset.
func GenerateModuleIncludeListGo() error {
	const moduleIncludeListCmdPath = "dev-tools/cmd/module_include_list/module_include_list.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	licenseType := BeatLicense
	if licenseType == "ASL 2.0" {
		licenseType = "ASL2"
	}

	moduleFieldsCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, moduleIncludeListCmdPath),
		"-license", licenseType,
		filepath.Join(CWD(), "module"),
	)

	return moduleFieldsCmd()
}
