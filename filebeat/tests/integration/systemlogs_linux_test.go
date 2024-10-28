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
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

//go:embed testdata/filebeat_system_module.yml
var systemModuleCfg string

// TestSystemLogsCanUseJournald aims to ensure the system-logs input can
// correctly choose and start a journald input when the globs defined in
// var.paths do not resolve to any file.
func TestSystemLogsCanUseJournaldInput(t *testing.T) {
	skipOnBuildKite(t)
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

func waitForAllFilesets(t *testing.T, outputGlob string, msgAndArgs ...any) {
	require.Eventually(
		t,
		findFilesetNames(t, outputGlob),
		time.Minute,
		10*time.Millisecond,
		msgAndArgs...)
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

func writeToFile(t *testing.T, data []byte, name string) {
	if err := os.MkdirAll(filepath.Join("../", "../", "build", "integration-tests"), 0750); err != nil {
		t.Errorf("cannot create dirs: %s", err)
		return
	}
	f, err := os.Create(filepath.Join("../", "../", "build", "integration-tests", name))
	if err != nil {
		t.Errorf("cannot create '%s': %s", name, err)
	}

	defer f.Close()

	if _, err := f.Write(data); err != nil {
		t.Errorf("cannot write to '%s': '%s'", name, err)
	}
}

func findFilesetNames(t *testing.T, outputGlob string) func() bool {
	f := func() bool {
		files, err := filepath.Glob(outputGlob)
		if err != nil {
			t.Fatalf("cannot get files list for glob '%s': '%s'", outputGlob, err)
		}

		if len(files) > 1 {
			t.Fatalf(
				"only a single output file is supported, found: %d. Files: %s",
				len(files),
				files,
			)
		}

		foundSyslog := false
		foundAuth := false

		file, err := os.Open(files[0])
		if err != nil {
			t.Fatalf("cannot open '%s': '%s'", files[0], err)
		}
		defer file.Close()

		r := bufio.NewReader(file)
		for {
			line, err := r.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				} else {
					t.Fatalf("cannot read '%s': '%s", file.Name(), err)
				}
			}

			data := struct {
				Fileset struct {
					Name string `json:"name"`
				} `json:"fileset"`
			}{}

			if err := json.Unmarshal(line, &data); err != nil {
				t.Fatalf("cannot parse output line as JSON: %s", err)
			}

			switch data.Fileset.Name {
			case "syslog":
				foundSyslog = true
			case "auth":
				foundAuth = true
			}

			if foundAuth && foundSyslog {
				return true
			}
		}

		return false
	}

	return f
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

func skipOnBuildKite(t *testing.T) {
	val, isSet := os.LookupEnv("BUILDKITE")
	if !isSet {
		// if the envvar BUILDKITE is not set, we're not on BuildKite,
		// so return false (do not skip any test)
		return
	}

	buildkite, err := strconv.ParseBool(val)
	if err != nil {
		t.Fatalf("cannot parse '%s' as bool: %s", val, err)
	}

	if !buildkite {
		// We're not on BuildKite, do not  skip any test
		return
	}

	// os.Geteuid() == 0 means we're root.
	// If we're not root, skip the test
	if os.Geteuid() != 0 {
		t.Skip("this test can only run on BuildKite as root")
	}
}
