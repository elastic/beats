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

package fields

import (
	"io/ioutil"
	"path/filepath"
)

var indentByModule = map[string]int{
	"processors": 0,
	"module":     8,
	"active":     8,
	"protos":     8,
}

// GetModules returns a the list of modules for the given modules directory
func GetModules(modulesDir string) ([]string, error) {
	moduleInfos, err := ioutil.ReadDir(modulesDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, info := range moduleInfos {
		if !info.IsDir() {
			continue
		}
		names = append(names, info.Name())
	}
	return names, nil
}

// CollectModuleFiles looks for fields.yml files under the
// specified modules directory
func CollectModuleFiles(modulesDir string) ([]*YmlFile, error) {
	modules, err := GetModules(modulesDir)
	if err != nil {
		return nil, err
	}

	var files []*YmlFile
	for _, m := range modules {
		f, err := CollectFiles(m, modulesDir)
		if err != nil {
			return nil, err
		}
		files = append(files, f...)
	}

	return files, nil
}

// CollectFiles collects all files for the given module including filesets
func CollectFiles(module string, modulesPath string) ([]*YmlFile, error) {
	var files []*YmlFile

	fieldsYmlPath := filepath.Join(modulesPath, module, "_meta/fields.yml")
	ymlFile, err := NewYmlFile(fieldsYmlPath, 0)

	if err != nil {
		return nil, err
	} else if ymlFile != nil {
		files = append(files, ymlFile)
	}

	modulesRoot := filepath.Base(modulesPath)
	sets, err := ioutil.ReadDir(filepath.Join(modulesPath, module))
	if err != nil {
		return nil, err
	}

	for _, s := range sets {
		if !s.IsDir() {
			continue
		}

		fieldsYmlPath = filepath.Join(modulesPath, module, s.Name(), "_meta/fields.yml")
		ymlFile, err := NewYmlFile(fieldsYmlPath, indentByModule[modulesRoot])

		if err != nil {
			return nil, err
		} else if ymlFile != nil {
			files = append(files, ymlFile)
		}
	}
	return files, nil
}
