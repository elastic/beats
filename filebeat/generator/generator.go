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

package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

// DirExists check that directory exists
func DirExists(dir string) bool {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return true
	}

	return false
}

// CreateDirectories create directories in baseDir
func CreateDirectories(baseDir string, directories ...string) error {
	for _, d := range directories {
		p := path.Join(baseDir, d)
		err := os.MkdirAll(p, 0o750)
		if err != nil {
			return err
		}
	}

	return nil
}

// CopyTemplates copy templates from source, make replacement in template content and save it to dest
func CopyTemplates(src, dest string, templates []string, replace map[string]string) error {
	for _, template := range templates {
		err := copyTemplate(path.Join(src, template), path.Join(dest, template), replace)
		if err != nil {
			return err
		}
	}

	return nil
}

// AppendTemplate read template, make replacement in content and append it to dest
func AppendTemplate(template, dest string, replace map[string]string) error {
	c, err := readTemplate(template, replace)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		_, err = f.Write(c)
	}
	if err != nil {
		return fmt.Errorf("cannot append template: %v", err)
	}

	return nil
}

func copyTemplate(template, dest string, replace map[string]string) error {
	c, err := readTemplate(template, replace)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dest, c, 0o644)
	if err != nil {
		return fmt.Errorf("cannot copy template: %v", err)
	}

	return nil
}

func readTemplate(template string, replace map[string]string) ([]byte, error) {
	c, err := ioutil.ReadFile(template)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot read template: %v", err)
	}

	for oldV, newV := range replace {
		c = bytes.Replace(c, []byte("{"+oldV+"}"), []byte(newV), -1)
	}

	return c, nil
}

// RenameConfigYml renames config.yml to the name of the fileset, otherwise Filebeat refuses to start
func RenameConfigYml(modulesPath, module, fileset string) error {
	old := path.Join(modulesPath, "module", module, fileset, "config", "config.yml")
	new := path.Join(modulesPath, "module", module, fileset, "config", fileset+".yml")

	return os.Rename(old, new)
}
