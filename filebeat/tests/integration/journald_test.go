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
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

//go:embed testdata/filebeat_journald.yml
var journaldInputCfg string

func TestJournaldInputRunsAndRecoversFromJournalctlFailures(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// render configuration
	syslogID := fmt.Sprintf("%s-%s", t.Name(), uuid.Must(uuid.NewV4()).String())
	yamlCfg := fmt.Sprintf(journaldInputCfg, syslogID, filebeat.TempDir())

	generateJournaldLogs(t, syslogID, 3, 100)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()
	filebeat.WaitLogsContains("journalctl started", 10*time.Second, "journalctl did not start")

	pidLine := filebeat.GetLastLogLine("journalctl started")
	logEntry := struct {
		Pid int `json:"process.pid"`
	}{}
	if err := json.Unmarshal([]byte(pidLine), &logEntry); err != nil {
		t.Errorf("could not parse PID log entry as JSON: %s. Line: %q", err, pidLine)
	}

	pid := logEntry.Pid
	filebeat.WaitPublishedEvents(5*time.Second, 3)

	// Kill journalctl
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("coluld not kill journalctl with PID %d: %s", pid, err)
	}

	generateJournaldLogs(t, syslogID, 5, 100)
	filebeat.WaitLogsContains("journalctl started", 10*time.Second, "journalctl did not start")
	filebeat.WaitPublishedEvents(5*time.Second, 8)
}

func TestJournaldInputDoesNotDuplicateData(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// render configuration
	syslogID := fmt.Sprintf("%s-%s", t.Name(), uuid.Must(uuid.NewV4()).String())
	yamlCfg := fmt.Sprintf(journaldInputCfg, syslogID, filebeat.TempDir())

	defer func() {
		if t.Failed() {
			t.Logf("Syslog ID: %q", syslogID)
		}
	}()
	generateJournaldLogs(t, syslogID, 3, 100)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()
	filebeat.WaitLogsContains("journalctl started", 10*time.Second, "journalctl did not start")

	// Ensure started line is logged and PID is correctly read
	pidLine := filebeat.GetLastLogLine("journalctl started")
	logEntry := struct {
		Pid int `json:"process.pid"`
	}{}
	if err := json.Unmarshal([]byte(pidLine), &logEntry); err != nil {
		t.Errorf("could not parse PID log entry as JSON: %s. Line: %q", err, pidLine)
	}

	filebeat.WaitPublishedEvents(5*time.Second, 3)

	// Stop Filebeat
	filebeat.Stop()

	// Generate more logs
	generateJournaldLogs(t, syslogID, 5, 100)
	// Restart Filebeat
	filebeat.Start()

	// Wait for journalctl to start
	filebeat.WaitLogsContains("journalctl started", 10*time.Second, "journalctl did not start")

	// Wait for last even in the output
	filebeat.WaitPublishedEvents(5*time.Second, 8)
}

func TestJournaldLargeLines(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// render configuration
	syslogID := fmt.Sprintf("%s-%s", t.Name(), uuid.Must(uuid.NewV4()).String())
	yamlCfg := fmt.Sprintf(journaldInputCfg, syslogID, filebeat.TempDir())

	defer func() {
		if t.Failed() {
			t.Logf("Syslog ID: %q", syslogID)
		}
	}()

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()

	evtLen := 9000
	generateJournaldLogs(t, syslogID, 5, evtLen)

	filebeat.WaitPublishedEvents(20*time.Second, 5)
	type evt struct {
		Message string `json:"message"`
	}

	evts := integration.GetEventsFromFileOutput[evt](filebeat, 5, false)
	for i, e := range evts {
		if len(e.Message) != evtLen {
			t.Errorf("event %d: expecting len %d, got %d", i, evtLen, len(e.Message))
			if len(e.Message) < 100 {
				t.Logf("Message: %q", e.Message)
			}
		}
	}
}

func generateJournaldLogs(t *testing.T, syslogID string, lines, size int) {
	cmd := exec.Command("systemd-cat", "-t", syslogID)
	w, err := cmd.StdinPipe()
	if err != nil {
		t.Errorf("cannot get stdin pipe from systemd-cat: %s", err)
	}
	if err := cmd.Start(); err != nil {
		t.Errorf("cannot start 'systemd-cat': %s", err)
	}
	defer func() {
		// Make sure systemd-cat terminates successfully so the messages
		// are correctly written to the journal
		if err := cmd.Wait(); err != nil {
			t.Errorf("error waiting for system-cat to finish: %s", err)
		}

		if !cmd.ProcessState.Success() {
			t.Errorf("systemd-cat exited with %d", cmd.ProcessState.ExitCode())
		}
	}()

	for range lines {
		expectedBytes := size + 1
		written, err := fmt.Fprintln(w, largeStr(t, size))
		if err != nil {
			t.Errorf("could not write message to journald: %s", err)
		}
		if written != expectedBytes {
			t.Errorf("could not write the whole message, expecing to write %d bytes, but wrote %d", expectedBytes, written)
		}
		time.Sleep(time.Millisecond)
	}

	if err := w.Close(); err != nil {
		t.Errorf("could not close stdin from systemd-cat, messages are likely not written to the  journal: %s", err)
	}
}

func largeStr(t *testing.T, len int) string {
	str := strings.Builder{}
	for range len {
		c := rand.Int32N(93) + 33
		if err := str.WriteByte(byte(c)); err != nil {
			t.Fatal(err)
		}
	}

	gen := str.String()
	return gen
}
