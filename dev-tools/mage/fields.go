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

	"github.com/pkg/errors"

	"github.com/magefile/mage/sh"
)

// GenerateFieldsYAML generates a fields.yml file for a Beat. This will include
// the common fields specified by libbeat, the common fields for the Beat,
// and any additional fields.yml files you specify.
//
// moduleDirs specifies additional directories to search for modules. The
// contents of each fields.yml will be included in the generated file.
func GenerateFieldsYAML(moduleDirs ...string) error {
	return generateFieldsYAML(OSSBeatDir(), "fields.yml", moduleDirs...)
}

// GenerateFieldsYAMLTo generates a YAML file containing the field definitions
// for the Beat. It's the same as GenerateFieldsYAML but with a configurable
// output file.
func GenerateFieldsYAMLTo(output string, moduleDirs ...string) error {
	return generateFieldsYAML(OSSBeatDir(), output, moduleDirs...)
}

func generateFieldsYAML(baseDir, output string, moduleDirs ...string) error {
	const globalFieldsCmdPath = "libbeat/scripts/cmd/global_fields/main.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	globalFieldsCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, globalFieldsCmdPath),
		"-es_beats_path", beatsDir,
		"-beat_path", baseDir,
		"-out", output,
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

	assetCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, assetCmdPath),
		"-pkg", "include",
		"-in", fieldsYML,
		"-out", createDir(out),
		"-license", toLibbeatLicenseName(BeatLicense),
		BeatName,
	)

	return assetCmd()
}

// GenerateModuleFieldsGo generates a fields.go file containing a copy of the
// each module's field.yml data in a format that can be embedded in Beat's
// binary.
func GenerateModuleFieldsGo(moduleDir string) error {
	const moduleFieldsCmdPath = "dev-tools/cmd/module_fields/module_fields.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	moduleFieldsCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, moduleFieldsCmdPath),
		"-beat", BeatName,
		"-license", toLibbeatLicenseName(BeatLicense),
		filepath.Join(moduleDir),
	)

	return moduleFieldsCmd()
}

// GenerateModuleIncludeListGo generates an include/list.go file containing
// a import statement for each module and dataset.
func GenerateModuleIncludeListGo() error {
	return GenerateIncludeListGo(nil, []string{
		filepath.Join(CWD(), "module"),
	})
}

// GenerateIncludeListGo generates an include/list.go file containing imports
// for the packages that match the paths (or globs) in importDirs (optional)
// and moduleDirs (optional).
func GenerateIncludeListGo(importDirs []string, moduleDirs []string) error {
	const moduleIncludeListCmdPath = "dev-tools/cmd/module_include_list/module_include_list.go"

	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	includeListCmd := sh.RunCmd("go", "run",
		filepath.Join(beatsDir, moduleIncludeListCmdPath),
		"-license", toLibbeatLicenseName(BeatLicense),
	)

	var args []string
	for _, dir := range importDirs {
		args = append(args, "-import", dir)
	}
	for _, dir := range moduleDirs {
		args = append(args, "-moduleDir", dir)
	}

	return includeListCmd(args...)
}

// toLibbeatLicenseName translates the license type used in packages to
// the identifiers used by github.com/elastic/beatslibbeat/licenses.
func toLibbeatLicenseName(name string) string {
	switch name {
	case "ASL 2.0":
		return "ASL2"
	case "Elastic License":
		return "Elastic"
	default:
		panic(errors.Errorf("invalid license name '%v'", name))
	}
}
