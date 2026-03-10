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

// This file was contributed to by generative AI

//go:build integration && linux

package integration

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

//go:embed testdata/filebeat_journald.yml
var journaldInputCfg string

//go:embed testdata/filebeat_journald_all_boots.yml
var journaldInputAllBootsCfg string

var bootListLineRE = regexp.MustCompile(`^\s*([+-]?\d+)\s+([0-9a-fA-F]{32})\s+(.+)$`)

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

type bootInfo struct {
	Offset         string
	BootID         string
	StartTimestamp string
}

type journaldAllBootsEvent struct {
	Journald struct {
		Host struct {
			BootID string `json:"boot_id"`
		} `json:"host"`
	} `json:"journald"`
}

func TestJournaldInputReadsMessagesFromAllBoots(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	t.Log("Reading boot entries.")
	boots, listBootsRaw := listBoots(t)
	if len(boots) <= 1 {
		t.Fatalf("expected more than one boot in journalctl --list-boots output, got %d. Output:\n%s", len(boots), listBootsRaw)
	}

	oldestBoot := boots[0]
	secondOldestBoot := boots[1]

	t.Log("Counting boot entries: ", oldestBoot.Offset, oldestBoot.BootID)
	oldestBootEntries := countBootEntries(t, oldestBoot.Offset)

	if oldestBootEntries > 50_000 {
		t.Skipf("Too many entries in the first boot %d > 50_000", oldestBootEntries)
	}

	t.Log("Counting second old boot entries: ", secondOldestBoot.Offset, secondOldestBoot.BootID)
	secondOldestBootEntries := countBootEntries(t, secondOldestBoot.Offset)

	if oldestBootEntries > 50_000 {
		t.Skipf("Too many entries in the second boot %d > 50_000", secondOldestBootEntries)
	}

	expectedMessages := oldestBootEntries + secondOldestBootEntries
	if expectedMessages < 1 {
		t.Fatalf(
			"expected at least one journal entry from oldest two boots, got 0 (oldest=%d second_oldest=%d)",
			oldestBootEntries,
			secondOldestBootEntries,
		)
	}

	yamlCfg := fmt.Sprintf(journaldInputAllBootsCfg, filebeat.TempDir())
	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()
	filebeat.WaitLogsContains("journalctl started", 10*time.Second, "journalctl did not start")

	waitForAtLeastPublishedEvents(t, filebeat, expectedMessages, 10*time.Minute)

	events := integration.GetEventsFromFileOutput[journaldAllBootsEvent](filebeat, expectedMessages, false)
	bootIDs := distinctBootIDs(events)
	assert.GreaterOrEqualf(
		t,
		len(bootIDs),
		2,
		"expected at least 2 distinct boot IDs; got %d. sample=%v",
		len(bootIDs),
		sortedBootIDsSample(bootIDs, 10),
	)

	if t.Failed() {
		t.Logf("journalctl --list-boots output:\n%s", listBootsRaw)
		t.Logf("oldest boot: offset=%s id=%s start=%q", oldestBoot.Offset, oldestBoot.BootID, oldestBoot.StartTimestamp)
		t.Logf("second oldest boot: offset=%s id=%s start=%q", secondOldestBoot.Offset, secondOldestBoot.BootID, secondOldestBoot.StartTimestamp)
		t.Logf("entry counts: oldest=%d second_oldest=%d expected_messages=%d", oldestBootEntries, secondOldestBootEntries, expectedMessages)
		t.Logf("events read=%d distinct_boot_ids=%d sample=%v", len(events), len(bootIDs), sortedBootIDsSample(bootIDs, 10))
	}
}

func listBoots(t *testing.T) (boots []bootInfo, raw string) {
	cmd := exec.Command("journalctl", "--list-boots", "--no-pager", "--quiet")
	output, err := cmd.CombinedOutput()
	raw = string(output)
	if err != nil {
		t.Fatalf("could not run journalctl --list-boots: %s. Output: %q", err, strings.TrimSpace(raw))
	}

	lines := strings.SplitSeq(raw, "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		matches := bootListLineRE.FindStringSubmatch(trimmed)
		if len(matches) != 4 {
			t.Fatalf("unexpected line in journalctl --list-boots output: %q", trimmed)
		}

		timeRange := strings.TrimSpace(matches[3])
		startTimestamp := timeRange
		if parts := strings.SplitN(timeRange, "—", 2); len(parts) == 2 {
			startTimestamp = strings.TrimSpace(parts[0])
		} else if parts := strings.SplitN(timeRange, "--", 2); len(parts) == 2 {
			startTimestamp = strings.TrimSpace(parts[0])
		}

		boots = append(boots, bootInfo{
			Offset:         matches[1],
			BootID:         strings.ToLower(matches[2]),
			StartTimestamp: startTimestamp,
		})
	}

	return boots, raw
}

func countBootEntries(t *testing.T, bootOffset string) int {
	cmd := exec.Command(
		"bash",
		"-c",
		`set -o pipefail; journalctl -b "$1" --output=json --no-pager --quiet | wc -l`,
		"countBootEntries",
		bootOffset,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("journalctl | wc -l failed for boot %q: %s. output=%q", bootOffset, err, strings.TrimSpace(string(output)))
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		t.Fatalf("could not parse wc -l output for boot %q: output=%q error=%s", bootOffset, countStr, err)
	}

	return count
}

func waitForAtLeastPublishedEvents(t *testing.T, b *integration.BeatProc, min int, timeout time.Duration) {
	if min < 1 {
		t.Fatalf("minimum number of events to wait for must be at least 1, got %d", min)
	}

	// The size limit breaks it for real-world usage or machines with large messages
	outputGlob := filepath.Join(b.TempDir(), "output-*.ndjson")
	const progressStep = 20_000
	nextProgressLog := progressStep

	logProgress := func(got int) {
		for got >= nextProgressLog {
			t.Logf("published events found: >=%d (current=%d) %.2f%% from total", nextProgressLog, got, float64(got)/float64(min))
			nextProgressLog += progressStep
		}
	}

	got := b.CountFileLines(outputGlob)
	if got >= min {
		return
	}
	logProgress(got)

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		got := b.CountFileLines(outputGlob)
		logProgress(got)
		assert.GreaterOrEqualf(collect, got, min, "expected at least %d events, got %d", min, got)
	}, timeout, 200*time.Millisecond)
}

func distinctBootIDs(events []journaldAllBootsEvent) map[string]struct{} {
	ids := make(map[string]struct{}, len(events))
	for _, evt := range events {
		bootID := strings.TrimSpace(evt.Journald.Host.BootID)
		if bootID == "" {
			continue
		}

		ids[bootID] = struct{}{}
	}

	return ids
}

func sortedBootIDsSample(bootIDs map[string]struct{}, max int) []string {
	if max <= 0 || len(bootIDs) == 0 {
		return nil
	}

	ids := make([]string, 0, len(bootIDs))
	for id := range bootIDs {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	if len(ids) > max {
		return ids[:max]
	}

	return ids
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
