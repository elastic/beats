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
	"os"
	"strings"
)

// TODO handle errors
func (d *Diagnostics) getBeatInfo() {
	fmt.Fprintf(os.Stdout, "Retrieving beats metadata\n")
	response := d.apiRequest("/state")
	bs, _ := json.Marshal(response)
	json.Unmarshal(bs, &d.Beat.State)
	beatall, _ := json.Marshal(&d.Beat)
	d.writeToFile(d.DiagFolder, "beat.json", beatall)
}

// TODO, certain fields should be anonymized here.
// TODO, filebeat is hardcoded, needs to support all beats.
func (d *Diagnostics) copyBeatConfig() {
	fmt.Fprintf(os.Stdout, "Copying beats configuration files\n")
	srcpath := fmt.Sprintf("%s/filebeat.yml", d.Beat.ConfigPath)
	dstpath := fmt.Sprintf("%s/filebeat.yml", d.DiagFolder)
	d.copyFiles(srcpath, dstpath)
}

func (d *Diagnostics) copyModuleConfig() {
	fmt.Fprintf(os.Stdout, "Copying modules configuration files\n")
	fds, err := ioutil.ReadDir(d.Beat.ModulePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying modules configuration files %s\n", err)
	}
	for _, fd := range fds {
		if strings.HasSuffix(fd.Name(), ".yml") {
			srcpath := fmt.Sprintf("%s/%s", d.Beat.ModulePath, fd.Name())
			dstpath := fmt.Sprintf("%s/%s", d.DiagFolder, fd.Name())
			d.copyFiles(srcpath, dstpath)
		}
	}
}

// TODO, Currently hardcoded to filebeat, needs to change based on beat type.
func (d *Diagnostics) copyBeatLogs() {
	fmt.Fprintf(os.Stdout, "Copying beats logs\n")
	srcpath := fmt.Sprintf("%s/filebeat", d.Beat.LogPath)
	dstpath := fmt.Sprintf("%s/filebeat.log", d.DiagFolder)
	d.copyFiles(srcpath, dstpath)
}

func (d *Diagnostics) createManifest() {
	d.Manifest.Command = d.Type
	d.Manifest.Version = d.Beat.Info.Version
	manifest, _ := json.Marshal(&d.Manifest)
	d.writeToFile(d.DiagFolder, "manifest.json", manifest)
}
