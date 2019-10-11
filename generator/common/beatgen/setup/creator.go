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

package setup

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/pkg/errors"
)

// CfgPrefix specifies the env variable prefix used to configure the beat
const CfgPrefix = "NEWBEAT"

// GenNewBeat generates a new custom beat
// We assume our config object is populated and valid here
func GenNewBeat(config map[string]string) error {
	if config["type"] != "beat" && config["type"] != "metricbeat" {
		return fmt.Errorf("%s is not a valid custom beat type. Valid types are 'beat' and 'metricbeat'", config["type"])
	}

	genPath := devtools.OSSBeatDir("generator", config["type"], "{beat}")
	err := filepath.Walk(genPath, func(path string, info os.FileInfo, err error) error {
		newBase := filepath.Join(build.Default.GOPATH, "src", config["beat_path"])
		replacePath := strings.Replace(path, genPath, newBase, -1)

		writePath := strings.Replace(replacePath, "{beat}", config["project_name"], -1)
		writePath = strings.Replace(writePath, ".go.tmpl", ".go", -1)
		if info.IsDir() {
			err := os.MkdirAll(writePath, 0755)
			if err != nil {
				return errors.Wrapf(err, "error creating directory %s", writePath)
			}
		} else {

			//dump original source file
			tmplFile, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "error reading source templatse file")
			}
			newFile := replaceVars(config, string(tmplFile))

			err = ioutil.WriteFile(writePath, []byte(newFile), 0644)
			if err != nil {
				return errors.Wrap(err, "error writing beat file")
			}
		}

		return nil
	})

	return err
}

// replaceVars replaces any template vars in a target file
// We're not using the golang template engine as it seems a tad heavy-handed for this use case
// We have a dozen or so files across various languages (go, make, etc) and most just need one or two vars replaced.
func replaceVars(config map[string]string, fileBody string) string {
	var newBody = fileBody
	config["beat"] = strings.ToLower(config["project_name"])
	for tmplName, tmplValue := range config {
		tmplStr := fmt.Sprintf("{%s}", tmplName)
		newBody = strings.ReplaceAll(newBody, tmplStr, tmplValue)
	}

	return newBody
}
