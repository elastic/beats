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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var modulesDConfigTemplate = `
# Module: {{.Module}}
# Docs: https://www.elastic.co/guide/en/beats/{{.BeatName}}/{{ beat_doc_branch }}/{{.BeatName}}-module-{{.Module}}.html

{{.Config}}`[1:]

// GenerateDirModulesD generates a modules.d directory containing the
// <module>.yml.disabled files. It adds a header to each file containing a
// link to the documentation.
func GenerateDirModulesD() error {
	if err := os.RemoveAll("modules.d"); err != nil {
		return err
	}

	shortConfigs, err := filepath.Glob("module/*/_meta/config.yml")
	if err != nil {
		return err
	}

	for _, f := range shortConfigs {
		parts := strings.Split(filepath.ToSlash(f), "/")
		if len(parts) < 2 {
			continue
		}
		moduleName := parts[1]

		config, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		data, err := Expand(modulesDConfigTemplate, map[string]interface{}{
			"Module": moduleName,
			"Config": string(config),
		})
		if err != nil {
			return err
		}

		target := filepath.Join("modules.d", moduleName+".yml.disabled")
		err = ioutil.WriteFile(createDir(target), []byte(data), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

type datasetDefinition struct {
	Enabled *bool
}

type moduleDefinition struct {
	Name     string                       `yaml:"module"`
	Filesets map[string]datasetDefinition `yaml:",inline"`
}

// ValidateDirModulesD validates a modules.d directory containing the
// <module>.yml.disabled files. It checks that the files are valid
// yaml and conform to module definitions.
func ValidateDirModulesD() error {
	_, err := loadModulesD()
	return err
}

// ValidateDirModulesDDatasetsDisabled ensures that all the datasets
// are disabled by default.
func ValidateDirModulesDDatasetsDisabled() error {
	cfgs, err := loadModulesD()
	if err != nil {
		return err
	}
	var errs multierror.Errors
	for path, cfg := range cfgs {
		// A config.yml is a list of module configurations.
		for modIdx, mod := range cfg {
			// A module config is a map of datasets.
			for dsName, ds := range mod.Filesets {
				if ds.Enabled == nil || *ds.Enabled {
					var entry string
					if len(cfg) > 1 {
						entry = fmt.Sprintf(" (entry #%d)", modIdx+1)
					}
					err = fmt.Errorf("in file '%s': %s module%s dataset %s must be explicitly disabled (needs `enabled: false`)",
						path, mod.Name, entry, dsName)
					errs = append(errs, err)
				}
			}
		}
	}
	return errs.Err()
}

func loadModulesD() (modules map[string][]moduleDefinition, err error) {
	files, err := filepath.Glob("modules.d/*.disabled")
	if err != nil {
		return nil, err
	}
	modules = make(map[string][]moduleDefinition, len(files))
	for _, file := range files {
		contents, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "reading %s", file)
		}
		var cfg []moduleDefinition
		if err = yaml.Unmarshal(contents, &cfg); err != nil {
			return nil, errors.Wrapf(err, "parsing %s as YAML", file)
		}
		modules[file] = cfg
	}
	return modules, nil
}
