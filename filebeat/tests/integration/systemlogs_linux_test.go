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
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

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
	yamlCfg := fmt.Sprintf(systemModuleCfg, globWithoutFiles, globWithoutFiles, workDir)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start(
		"-E",
		"logging.event_data.files.rotateeverybytes=524288000",
	)

	filebeat.WaitForLogs(
		"no files were found, using journald input",
		10*time.Second,
		"system-logs did not select journald input")
	filebeat.WaitForLogs(
		"journalctl started with PID",
		10*time.Second,
		"system-logs did not start journald input")

	// Scan every event in the output until at least one from
	// each fileset (auth, syslog) is found.
	waitForAllFilesets(
		t,
		filepath.Join(workDir, "output*.ndjson"),
		"did not find events from both filesets: 'auth' and 'syslog'",
	)
}

func TestDebugBuildKite(t *testing.T) {
	jctlSyslog := exec.Command("journalctl",
		"--utc",
		"--output", "json",
		"--no-pager",
		"--facility", "0",
		"--facility", "1",
		"--facility", "2",
		"--facility", "3",
		"--facility", "5",
		"--facility", "6",
		"--facility", "7",
		"--facility", "8",
		"--facility", "9",
		"--facility", "11",
		"--facility", "12",
		"--facility", "15",
		"-n", "5")

	syslogOut, err := jctlSyslog.CombinedOutput()
	if err != nil {
		t.Errorf("cannot run journalctl for syslog: %s", err)
	}
	writeToFile(t, syslogOut, "syslogOut")

	jctlAuth := exec.Command("journalctl",
		"--utc",
		"--output", "json",
		"--no-pager",
		"--facility", "4",
		"--facility", "10",
		"-n", "5")
	authOut, err := jctlAuth.CombinedOutput()
	if err != nil {
		t.Errorf("cannot run journalctl for auth: %s", err)
	}
	writeToFile(t, authOut, "authOut")

	cmds := []string{"whoami", "groups"}
	for _, cmd := range cmds {
		c := exec.Command(cmd)
		out, err := c.CombinedOutput()
		if err != nil {
			t.Errorf("cannot execute '%s': '%s'", cmd, err)
			continue
		}
		writeToFile(t, out, cmd)
	}
}
