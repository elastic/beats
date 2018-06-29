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
	"os"
	"path/filepath"
)

var indentByModule = map[string]int{
	"processors": 0,
	"module":     8,
	"active":     8,
	"protos":     8,
}

// CollectModuleFiles looks for fields.yml files under the
// specified root directory
func CollectModuleFiles(root string) ([]*YmlFile, error) {
	modules, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var files []*YmlFile
	for _, m := range modules {
		f, err := collectFiles(m, root)
		if err != nil {
			return nil, err
		}
		files = append(files, f...)
	}

	return files, nil
}

func collectFiles(module os.FileInfo, modulesPath string) ([]*YmlFile, error) {
	if !module.IsDir() {
		return nil, nil
	}

	var files []*YmlFile
	fieldsYmlPath := filepath.Join(modulesPath, module.Name(), "_meta", "fields.yml")
	if _, err := os.Stat(fieldsYmlPath); !os.IsNotExist(err) {
		files = append(files, &YmlFile{
			Path:   fieldsYmlPath,
			Indent: 0,
		})
	} else if !os.IsNotExist(err) && err != nil {
		return nil, err
	}

	modulesRoot := filepath.Base(modulesPath)
	sets, err := ioutil.ReadDir(filepath.Join(modulesPath, module.Name()))
	if err != nil {
		return nil, err
	}

	for _, s := range sets {
		if !s.IsDir() {
			continue
		}

		fieldsYmlPath = filepath.Join(modulesPath, module.Name(), s.Name(), "_meta", "fields.yml")
		if _, err = os.Stat(fieldsYmlPath); !os.IsNotExist(err) {
			files = append(files, &YmlFile{
				Path:   fieldsYmlPath,
				Indent: indentByModule[modulesRoot],
			})
		} else if !os.IsNotExist(err) && err != nil {
			return nil, err
		}
	}
	return files, nil
}
