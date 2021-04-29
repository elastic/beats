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

package diagnostics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

func getBeatInfo(diag *Diagnostics) {
	diag.Logger.Info("Gathering beats metadata")
	bjson, err := json.Marshal(diag.Beat)
	if err != nil {
		diag.Logger.Error("Failed to marshal beats information")
		fmt.Println(err)
	}
	writeToFile(diag.DiagFolder, "beat.json", bjson)
}

func copyBeatConfig(diag *Diagnostics) {
	diag.Logger.Info("Copying beats configuration files")
	srcpath := fmt.Sprintf("%s/filebeat.yml", diag.Beat.ConfigPath)
	dstpath := fmt.Sprintf("%s/filebeat.yml", diag.DiagFolder)
	copyFiles(srcpath, dstpath)
}

func copyModuleConfig(diag *Diagnostics) {
	diag.Logger.Info("Copying modules configuration files")
	fds, err := ioutil.ReadDir(diag.Beat.ModulePath)
	if err != nil {
		diag.Logger.Error("Error copying modules configuration files", err)
	}
	for _, fd := range fds {
		if strings.HasSuffix(fd.Name(), ".yml") {
			srcpath := fmt.Sprintf("%s/%s", diag.Beat.ModulePath, fd.Name())
			dstpath := fmt.Sprintf("%s/%s", diag.DiagFolder, fd.Name())
			copyFiles(srcpath, dstpath)
		}
	}
}

func copyBeatLogs(diag *Diagnostics) {
	diag.Logger.Info("Copying beats logs")
	srcpath := fmt.Sprintf("%s/filebeat", diag.Beat.LogPath)
	dstpath := fmt.Sprintf("%s/filebeat.log", diag.DiagFolder)
	copyFiles(srcpath, dstpath)
}
