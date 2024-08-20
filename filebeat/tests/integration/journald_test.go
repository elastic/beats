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
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func generateJournaldLogs(t *testing.T, ctx context.Context, syslogID string, max int) {
	cmd := exec.Command("systemd-cat", "-t", syslogID)
	w, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("cannot get stdin pipe from systemd-cat: %s", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("cannot start 'systemd-cat': %s", err)
	}
	defer func() {
		if err := cmd.Wait(); err != nil {
			t.Errorf("error waiting for system-cat to finish: %s", err)
		}

		fmt.Println("Success?", cmd.ProcessState.Success(), "Exit code:", cmd.ProcessState.ExitCode())
	}()

	for count := 1; count <= max; count++ {
		i, err := fmt.Fprintf(w, "Count: %03d\n", count)
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>", i, "bytes written", err)
		if err != nil {
			t.Errorf("could not write message to journald: %s", err)
		}
		time.Sleep(time.Millisecond)
	}

	fmt.Println("closing stdin:", w.Close())
}

//go:embed testdata/filebeat_journald.yml
var journaldInputCfg string

func TestJournaldInput(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// render configuration
	syslogID := fmt.Sprintf("%s-%s", t.Name(), uuid.New().String())
	yamlCfg := fmt.Sprintf(journaldInputCfg, syslogID, filebeat.TempDir())

	go generateJournaldLogs(t, context.Background(), syslogID, 3)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()
	filebeat.WaitForLogs("journalctl started with PID", 10*time.Second, "journalctl did not start")

	pidLine := filebeat.GetLogLine("journalctl started with PID")
	logEntry := struct{ Message string }{}
	if err := json.Unmarshal([]byte(pidLine), &logEntry); err != nil {
		t.Fatalf("could not parse PID log entry as JSON: %s", err)
	}

	pid := 0
	fmt.Sscanf(logEntry.Message, "journalctl started with PID %d", &pid)

	filebeat.WaitForLogs("Count: 003", 5*time.Second, "did not find the third event in published events")

	// Kill journalctl
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("coluld  not kill journalctl with PID %d: %s", pid, err)
	}

	go generateJournaldLogs(t, context.Background(), syslogID, 5)
	filebeat.WaitForLogs("journalctl started with PID", 10*time.Second, "journalctl did not start")
	filebeat.WaitForLogs("Count: 005", time.Second, "expected log message not found in published events SECOND")

	eventsPublished := filebeat.CountFileLines(filepath.Join(filebeat.TempDir(), "output-*.ndjson"))

	if eventsPublished != 8 {
		t.Fatalf("expecting 8 published events, got %d instead'", eventsPublished)
	}
}
