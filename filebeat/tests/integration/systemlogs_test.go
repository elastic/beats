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
	"path"
	"path/filepath"
	"testing"
	"time"

	cp "github.com/otiai10/copy"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

//go:embed testdata/filebeat_system_module.yml
var systemModuleCfg string

// TestSystemLogsCanUseJournald aims to ensure the system-logs input can
// correctly choose and start a journald input when the globs defined in
// var.paths do not resolve to any file.
func TestSystemLogsCanUseJournaldInput(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()
	copyModulesDir(t, workDir)

	// As the name says, we want this folder to exist bu t be empty
	globWithoutFiles := filepath.Join(filebeat.TempDir(), "this-folder-does-not-exist")
	yamlCfg := fmt.Sprintf(systemModuleCfg, globWithoutFiles, workDir)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()

	filebeat.WaitForLogs(
		"no files were found, using journald input",
		10*time.Second,
		"system-logs did not select journald input")
	filebeat.WaitForLogs(
		"journalctl started with PID",
		10*time.Second,
		"system-logs did not start journald input")
}

func TestSystemLogsCanUseLogInput(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()
	copyModulesDir(t, workDir)

	logFilePath := path.Join(workDir, "syslog")
	integration.GenerateLogFile(t, logFilePath, 5, false)
	yamlCfg := fmt.Sprintf(systemModuleCfg, logFilePath, workDir)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()

	filebeat.WaitForLogs(
		"using log input because file(s) was(were) found",
		10*time.Second,
		"system-logs did not select the log input")
	filebeat.WaitForLogs(
		"Harvester started for paths:",
		10*time.Second,
		"system-logs did not start the log input")
}

func copyModulesDir(t *testing.T, dst string) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get the current directory: %s", err)
	}
	localModules := filepath.Join(pwd, "../", "../", "module")
	localModulesD := filepath.Join(pwd, "../", "../", "modules.d")

	if err := cp.Copy(localModules, filepath.Join(dst, "module")); err != nil {
		t.Fatalf("cannot copy 'module' folder to test folder: %s", err)
	}
	if err := cp.Copy(localModulesD, filepath.Join(dst, "modules.d")); err != nil {
		t.Fatalf("cannot copy 'modules.d' folder to test folder: %s", err)
	}
}
