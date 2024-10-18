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

//go:build integration && linux

package integration

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

//go:embed testdata/filebeat_system_module.yml
var systemModuleCfg string

// TestSystemLogsCanUseJournald aims to ensure the system-logs input can
// correctly choose and start a journald input when the globs defined in
// var.paths do not resolve to any file.
func TestSystemLogsCanUseJournald(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// As the name says, we want this folder to exist bu t be empty
	emptyTempFolder := t.TempDir()
	yamlCfg := fmt.Sprintf(systemModuleCfg, emptyTempFolder, filebeat.TempDir())

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal("cannot get the current directory: %s", err)
	}
	localModules := os.DirFS(filepath.Join(pwd, "../", "../", "module"))
	localModulesD := os.DirFS(filepath.Join(pwd, "../", "../", "modules.d"))

	if err := os.CopyFS(filepath.Join(filebeat.TempDir(), "module"), localModules); err != nil {
		t.Fatalf("cannot copy 'module' folder to test folder: %s", err)
	}
	if err := os.CopyFS(filepath.Join(filebeat.TempDir(), "modules.d"), localModulesD); err != nil {
		t.Fatalf("cannot copy 'modules.d' folder to test folder: %s", err)
	}

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()
	filebeat.WaitForLogs(
		"journalctl started with PID",
		10*time.Second,
		"system-logs did not start journald")
}
