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
	"regexp"
	"strings"
)

// CollectModules collects module configs to modules.d
func CollectModules() error {
	header := `# Module: %[1]s
# Docs: https://www.elastic.co/guide/en/beats/%[2]s/%[3]s/%[2]s-module-%[1]s.html

`
	r, err := regexp.Compile(`.+\.reference\.yml`)
	if err != nil {
		return err
	}

	beatName := os.Getenv("BEAT_NAME")
	docsBranch := os.Getenv("DOCS_BRANCH")

	path, err := filepath.Abs("module")
	if err != nil {
		return err
	}

	modules, err := ioutil.ReadDir("module")
	if err != nil {
		return err
	}

	if err = os.Mkdir("modules.d", os.ModeDir|os.ModePerm); err != nil {
		return err
	}

	modulesDDir, err := filepath.Abs("modules.d")
	if err != nil {
		return err
	}

	for _, module := range modules {
		moduleConfsGlob := filepath.Join(path, module.Name(), "_meta/config*.yml")
		moduleConfs, err := filepath.Glob(moduleConfsGlob)
		if err != nil {
			return err
		}

		for _, moduleConf := range moduleConfs {
			if r.MatchString(moduleConf) {
				continue
			}

			info, err := os.Stat(moduleConf)
			if err != nil {
				return err
			}
			if info.IsDir() {
				continue
			}

			moduleFile := fmt.Sprintf(header, module.Name(), beatName, docsBranch)
			disabledConfigFilename := strings.Replace(filepath.Base(moduleConf), "config", module.Name(), -1) + ".disabled"

			fileBytes, err := ioutil.ReadFile(moduleConf)
			if err != nil {
				return err
			}

			moduleFile += string(fileBytes)

			err = ioutil.WriteFile(filepath.Join(modulesDDir, disabledConfigFilename), []byte(moduleFile), 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
