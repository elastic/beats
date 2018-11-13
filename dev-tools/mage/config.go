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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const moduleConfigTemplate = `
#==========================  Modules configuration =============================
{{.BeatName}}.modules:
{{range $mod := .Modules}}
#{{$mod.Dashes}} {{$mod.Title | title}} Module {{$mod.Dashes}}
{{$mod.Config}}
{{- end}}

`

type moduleConfigTemplateData struct {
	ID     string
	Title  string
	Dashes string
	Config string
}

type moduleFieldsYmlData []struct {
	Title       string `json:"title"`
	ShortConfig bool   `json:"short_config"`
}

func readModuleFieldsYml(path string) (title string, useShort bool, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	var fd moduleFieldsYmlData
	if err = yaml.Unmarshal(data, &fd); err != nil {
		return "", false, err
	}

	if len(fd) == 0 {
		return "", false, errors.New("module not found in fields.yml")
	}

	return fd[0].Title, fd[0].ShortConfig, nil
}

// moduleDashes returns a string containing the correct number of dashes '-' to
// center the modules title in the middle of the line surrounded by an equal
// number of dashes on each side.
func moduleDashes(name string) string {
	const (
		lineLen        = 80
		headerLen      = len("#")
		titleSuffixLen = len(" Module ")
	)

	numDashes := lineLen - headerLen - titleSuffixLen - len(name) - 1
	numDashes /= 2
	return strings.Repeat("-", numDashes)
}

// GenerateModuleReferenceConfig generates a reference config file and includes
// modules found from the given module dirs.
func GenerateModuleReferenceConfig(out string, moduleDirs ...string) error {
	var moduleConfigs []moduleConfigTemplateData
	for _, dir := range moduleDirs {
		modules, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, modDirInfo := range modules {
			if !modDirInfo.IsDir() {
				continue
			}
			name := modDirInfo.Name()

			// Get title from fields.yml.
			title, _, err := readModuleFieldsYml(filepath.Join(dir, name, "_meta/fields.yml"))
			if err != nil {
				title = strings.Title(name)
			}

			// Prioritize config.reference.yml, but fallback to config.yml.
			files := []string{
				filepath.Join(dir, name, "_meta/config.reference.yml"),
				filepath.Join(dir, name, "_meta/config.yml"),
			}

			var data []byte
			for _, f := range files {
				data, err = ioutil.ReadFile(f)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return err
				}

				break
			}
			if data == nil {
				continue
			}

			moduleConfigs = append(moduleConfigs, moduleConfigTemplateData{
				ID:     name,
				Title:  title,
				Dashes: moduleDashes(title),
				Config: string(data),
			})
		}
	}

	// Sort them by their module dir name, but put system first.
	sort.Slice(moduleConfigs, func(i, j int) bool {
		// Bubble system to the top of the list.
		if moduleConfigs[i].ID == "system" {
			return true
		} else if moduleConfigs[j].ID == "system" {
			return false
		}
		return moduleConfigs[i].ID < moduleConfigs[j].ID
	})

	config := MustExpand(moduleConfigTemplate, map[string]interface{}{
		"Modules": moduleConfigs,
	})

	return ioutil.WriteFile(createDir(out), []byte(config), 0644)
}
