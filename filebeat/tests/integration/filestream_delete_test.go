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

//This file was contributed to by generative AI

//go:build integration

package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/mock-es/pkg/api"
)

var logFileLines = []string{
	"You can't connect the panel without connecting the wireless AGP panel!",
	"We need to back up the haptic FTP hard drive!",
	"Indexing the array won't do anything, we need to parse the neural SMTP system!",
	"I'll generate the haptic TCP pixel, that should transmitter the JBOD application!",
	"I'll quantify the wireless XSS driver, that should port the HTTP driver!",
	"If we connect the program, we can get to the ADP alarm through the back-end EXE pixel!",
	"I'll generate the primary SSL port, that should firewall the IB firewall!",
	"I'll program the digital RSS bus, that should sensor the JSON system!",
	"Hacking the feed won't do anything, we need to input the optical PNG microchip!",
	"We need to synthesize the solid state GB port!",
}

func TestFilestreamDelete(t *testing.T) {
	testCases := map[string]struct {
		configTmpl          string
		closeCondMsg        string
		resourceNotFinished bool
		dataAdded           bool
		gracePeriod         time.Duration
	}{
		"EOF": {
			configTmpl:   "eof.yml",
			closeCondMsg: "EOF has been reached. Closing. Path='%s'",
		},
		"EOF and resource not finished": {
			configTmpl:          "eof.yml",
			closeCondMsg:        "EOF has been reached. Closing. Path='%s'",
			resourceNotFinished: true,
		},
		"EOF resource not finished data added during grace priod": {
			configTmpl:          "eof.yml",
			closeCondMsg:        "EOF has been reached. Closing. Path='%s'",
			resourceNotFinished: true,
			dataAdded:           true,
			gracePeriod:         2 * time.Second,
		},
		"Inactive": {
			configTmpl:   "inactive.yml",
			closeCondMsg: "'%s' is inactive",
		},
		"Inactive and resource not finished": {
			configTmpl:          "inactive.yml",
			closeCondMsg:        "'%s' is inactive",
			resourceNotFinished: true,
		},
		"Inactive resource not finished and data added during grace period": {
			configTmpl:          "inactive.yml",
			closeCondMsg:        "'%s' is inactive",
			resourceNotFinished: true,
			dataAdded:           true,
			gracePeriod:         2 * time.Second,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			s, esAddr, es, _ := integration.StartMockES(t, "", 0, 0, 0, 0, 0)
			defer s.Close()

			if tc.resourceNotFinished {
				if err := es.UpdateOdds(0, 100, 0, 0); err != nil {
					t.Fatalf("cannot update odds from Mock-ES: %s", err)
				}
			}

			testDataPath, err := filepath.Abs("./testdata")
			if err != nil {
				t.Fatalf("cannot get absolute path for 'testdata': %s", err)
			}

			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)
			workDir := filebeat.TempDir()

			logFile := filepath.Join(workDir, "log.log")

			// Escape filepaths for Windows
			msgLogFilePath := logFile
			if runtime.GOOS == "windows" {
				msgLogFilePath = strings.ReplaceAll(logFile, `\`, `\\`)
			}
			integration.WriteLogFile(t, logFile, 100, false)

			vars := map[string]any{
				"homePath":    workDir,
				"logfile":     logFile,
				"testdata":    testDataPath,
				"esHost":      esAddr,
				"gracePeriod": tc.gracePeriod.String(),
			}
			cfgYAML := getConfig(t, vars, "delete", tc.configTmpl)
			filebeat.WriteConfigFile(cfgYAML)
			filebeat.Start()

			closeCondMsg := fmt.Sprintf(tc.closeCondMsg, msgLogFilePath)
			filebeat.WaitLogsContains(
				closeCondMsg,
				10*time.Second,
				"did not find close condition '%s' in the logs",
				closeCondMsg,
			)

			if tc.resourceNotFinished {
				testResourceNotFinished(t, filebeat, es, msgLogFilePath)
			}

			if tc.gracePeriod != 0 {
				// The grace period test also ensures the file has been
				// correctly removed
				testGracePeriod(
					t,
					filebeat,
					tc.gracePeriod,
					tc.dataAdded,
					msgLogFilePath)
			} else {
				// Wait for the file removed message
				removedMsg := fmt.Sprintf("'%s' removed", msgLogFilePath)
				filebeat.WaitLogsContains(
					removedMsg,
					30*time.Second,
					"file removed log entry not found")

				if fileExists(t, logFile) {
					t.Fatalf("%q should have been removed", logFile)
				}
			}
		})
	}
}

func testResourceNotFinished(
	t *testing.T,
	filebeat *integration.BeatProc,
	es *api.APIHandler,
	msgLogFilePath string) {

	t.Run("can detect events not published", func(t *testing.T) {
		// Wait a few times for the 'not finished' logs
		notFinishedMsg := fmt.Sprintf(
			"not all events from '%s' have been published, "+
				"closing harvester",
			msgLogFilePath)
		for i := range 2 {
			filebeat.WaitLogsContains(
				notFinishedMsg,
				10*time.Second,
				"[%d] Filebeat did not wait for the resource to be finished",
				i,
			)
		}
	})

	// Reset odds in Mock-ES
	if err := es.UpdateOdds(0, 0, 0, 0); err != nil {
		t.Fatalf("cannot update mock-es odds: %s", err)
	}
}

// testGracePeriod asserts:
// - Grace period is interrupted if data is added to the file
// - Grace period is respected if the file does not change
func testGracePeriod(
	t *testing.T,
	filebeat *integration.BeatProc,
	gracePeriod time.Duration,
	dataAdded bool,
	msgLogFilePath string) {

	if dataAdded {
		t.Run("grace period is interrupted when file changes", func(t *testing.T) {
			gracePeriodMsg := fmt.Sprintf(
				"all events from '%s' have been published, waiting for %s grace period",
				msgLogFilePath, gracePeriod)

			// This wait always takes 6.1s, I'm not quite sure why, probably it
			// is caused by some backoff logic. Setting the backoff.* in the
			// output is not enough. So we wait at least 10s here
			filebeat.WaitLogsContains(
				gracePeriodMsg,
				10*time.Second,
				"did not start waiting for grace period")

			// Wait 1/2 of the grace period, then add data to the file
			time.Sleep(gracePeriod / 2)
			integration.WriteLogFile(t, msgLogFilePath, 5, true)

			// Wait for the message of file size changed
			changedMsg := fmt.Sprintf("'%s' was updated, won't remove. Closing harvester", msgLogFilePath)
			filebeat.WaitLogsContains(changedMsg, time.Second, "filestream did detect the file change")

			// Make sure the harvester is closed
			// These two messages are logged at essentially the same time, so
			// their order in the log file is non-deterministic. Use
			// WaitLogsContainsAnyOrder to check for both without relying on order.
			filebeat.WaitLogsContainsAnyOrder(
				[]string{
					"Closing reader of filestream",
					"Stopped harvester for file",
				},
				5*time.Second,
				"harvester/reader was not closed")
		})
	}

	t.Run("grace period is respected", func(t *testing.T) {
		msg := fmt.Sprintf("'%s' removed", msgLogFilePath)
		filebeat.WaitLogsContains(msg, 30*time.Second, "file removed log entry not found")
		removedMsg := filebeat.GetLastLogLine(msg)

		gracePeriodMsg := fmt.Sprintf(
			"all events from '%s' have been published, waiting for %s grace period",
			msgLogFilePath, gracePeriod)
		beforeWait := filebeat.GetLastLogLine(gracePeriodMsg)

		delta := timeBetweenLogEntries(t, beforeWait, removedMsg)
		if delta < gracePeriod {
			t.Errorf("grace period of %s was not respected", gracePeriod)
			t.Log("grace period waiting calculated based on the following log entries:")
			t.Log("First :", beforeWait)
			t.Log("Second:", removedMsg)
		}
	})
}

// TestFilestreamDeleteEnabledOnExistingFiles tests the flow where Filestream
// has already ingested some files, then it is restarted with the delete
// feature enabled.
func TestFilestreamDeleteEnabledOnExistingFiles(t *testing.T) {
	testCases := map[string]struct {
		configTmpl          string
		msg                 string
		resourceNotFinished bool
		dataAdded           bool
		gracePeriod         time.Duration
	}{
		"EOF and grace priod": {
			configTmpl:  "restart-eof.yml",
			msg:         "EOF has been reached. Closing. Path='%s'",
			gracePeriod: 5 * time.Second,
		},
		"Inactive and grace period": {
			configTmpl:  "restart-inactive.yml",
			msg:         "'%s' is inactive",
			gracePeriod: 5 * time.Second,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			s, esAddr, _, _ := integration.StartMockES(t, "", 0, 0, 0, 0, 0)
			defer s.Close()

			testDataPath, err := filepath.Abs("./testdata")
			if err != nil {
				t.Fatalf("cannot get absolute path for 'testdata': %s", err)
			}

			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)
			workDir := filebeat.TempDir()

			logFile := filepath.Join(workDir, "log.log")
			// Escape filepaths for Windows
			msgLogFilePath := logFile
			if runtime.GOOS == "windows" {
				msgLogFilePath = strings.ReplaceAll(logFile, `\`, `\\`)
			}
			integration.WriteLogFile(t, logFile, 100, false)

			vars := map[string]any{
				"homePath":    workDir,
				"logfile":     logFile,
				"testdata":    testDataPath,
				"esHost":      esAddr,
				"gracePeriod": tc.gracePeriod.String(),
			}
			cfgYAML := getConfig(t, vars, "delete", tc.configTmpl)
			filebeat.WriteConfigFile(cfgYAML)
			filebeat.Start()

			msg := fmt.Sprintf(tc.msg, msgLogFilePath)
			filebeat.WaitLogsContains(
				msg,
				10*time.Second,
				"did not find '%s' in the logs",
				msg,
			)

			filebeat.Stop()
			filebeat.WaitLogsContains("filebeat stopped.", 2*time.Second, "Filebeat did not stop successfully")
			filebeat.RemoveLogFiles()

			if !fileExists(t, logFile) {
				t.Fatalf("%q should not have been removed", logFile)
			}

			vars["deleteEnabled"] = true
			cfgYAML = getConfig(t, vars, "delete", tc.configTmpl)
			filebeat.WriteConfigFile(cfgYAML)

			filebeat.Start()

			gracePeriodMsg := fmt.Sprintf(
				"all events from '%s' have been published, waiting for %s grace period",
				msgLogFilePath,
				tc.gracePeriod)
			filebeat.WaitLogsContains(gracePeriodMsg, 10*time.Second, "waiting for grace period log not found")

			msg = fmt.Sprintf("'%s' removed", msgLogFilePath)
			filebeat.WaitLogsContains(msg, 10*time.Second, "file removed log entry not found")

			if fileExists(t, logFile) {
				t.Fatalf("%q should have been removed", logFile)
			}
		})
	}
}

// TestFilestreamDeleteRealESFSAndNotify aims to simulate some real-world usage
// and test from the users' perspective. It is not an exhaustive test, nor does
// it aim to cover all scenarios. There are already extensive unit-tests.
//
// It uses a real Elasticsearch and queries data to ensure full ingestion
// while using fsnotify to monitor the target file for deletion.
func TestFilestreamDeleteRealESFSAndNotify(t *testing.T) {
	integration.EnsureESIsRunning(t)
	gracePeriod := 5 * time.Second
	delta := time.Second

	index := "test-delete" + uuid.Must(uuid.NewV4()).String()
	testDataPath, err := filepath.Abs("./testdata")
	if err != nil {
		t.Fatalf("cannot get absolute path for 'testdata': %s", err)
	}

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()

	logFile := filepath.Join(workDir, "log.log")
	logData := strings.Join(logFileLines[:5], "\n")
	logData += "\n" // Filebeat needs the '\n' to read the last line
	if err := os.WriteFile(logFile, []byte(logData), 0o644); err != nil {
		t.Fatalf("cannot write log file '%s': %s", logFile, err)
	}

	fileWatcher := integration.NewFileWatcher(t, logFile)
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			t.Errorf("File %s should not have been removed, removal happened at %s",
				event.Name,
				time.Now().Format(time.RFC3339Nano))
		}
	})
	fileWatcher.Start()
	defer fileWatcher.Stop()

	// We need the admin URL to create our custom index
	esURL := integration.GetESAdminURL(t, "http")

	// Create and start the proxy server
	proxy, proxyURL := integration.NewDisablingProxy(t, esURL.String())

	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	vars := map[string]any{
		"homePath":    workDir,
		"logfile":     logFile,
		"testdata":    testDataPath,
		"esHost":      proxyURL,
		"user":        user,
		"pass":        pass,
		"index":       index,
		"gracePeriod": gracePeriod.String(),
	}

	cfgYAML := getConfig(t, vars, "delete", "real-es.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	// Wait for data in ES
	msgs := []string{}
	require.Eventually(t, func() bool {
		msgs = integration.GetEventsMsgFromES(t, index, 200)
		return len(msgs) == len(logFileLines)/2
	}, time.Second*10, time.Millisecond*100, "not all log messages have been found on ES")

	// Wait for 1/2 of the grace period and add more data
	time.Sleep(gracePeriod / 2)

	// Add more data to the file
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("cannot open logfile to append data: %s", err)
	}
	logData2 := strings.Join(logFileLines[5:], "\n")
	logData2 += "\n"
	if _, err := f.WriteString(logData2); err != nil {
		t.Fatalf("could not append data to log file: %s", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("cannot flush log file: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("cannot close log file: %s", err)
	}

	// Disable (aka block) the output
	proxy.Disable()

	// Wait twice the grace period before unblocking the output
	blockedTimer := time.NewTimer(gracePeriod * 2)
	<-blockedTimer.C

	// Ensure log file still exists
	if !fileExists(t, logFile) {
		t.Fatal("file was removed while output was blocked")
	}

	// Unblock the output
	proxy.Enable()

	// Wait for the remaining data to be ingested
	msgs = []string{}
	require.Eventually(
		t,
		func() bool {
			msgs = integration.GetEventsMsgFromES(t, index, 200)
			return len(msgs) == len(logFileLines)
		},
		// This is the maximum time we will wait for the documents to
		// be query-able in Elasticserach. The documents might be fully
		// ingested and acknowledged by ES before we manage to query them,
		// hence this timeout might be equal or larger than the grace period.
		// If Filebeat deletes the file while we're wait for ES, the
		// fileWatcher will detect it and the registered callback will
		// fail the test.
		time.Second*3,
		time.Millisecond*100, "not all log messages have been found on ES")

	dataShippedTs := time.Now()
	fileRemovedChan := make(chan time.Time)
	// All events have been found, allow file to be removed
	// and get the removal timestamp
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			fileRemovedChan <- time.Now()
		}
	})

	deleteTimeout := gracePeriod * 3
	timeout := time.NewTimer(deleteTimeout)
	select {
	case fileRemovedTs := <-fileRemovedChan:
		timeElapsed := fileRemovedTs.Sub(dataShippedTs)
		// We need to use a delta here because there is a delay between
		// Filebeat receiving the last acknowledgement, thus starting to count
		// the grace period, and the test being able to access that all events
		// have been correctly ingested by Elasticsearch. We also query
		// Elasticsearch with an interval of 100ms, which only increases
		// the delay from when we capture 'dataShippedTs.'
		if timeElapsed < gracePeriod-delta {
			t.Fatalf("file was removed %s after data ingested (%s acceptable delta), but grace period was set to %s",
				timeElapsed,
				delta,
				gracePeriod)
		}
	case <-timeout.C:
		t.Fatalf("file was not removed within %d", deleteTimeout)
	}

	// Ensure the messages were ingested in the correct order
	allMesagesIngested(t, msgs, logFileLines)
}

func allMesagesIngested(t *testing.T, got, want []string) {
	t.Helper()

	for _, wantMsg := range want {
		found := false
		for _, gotMsg := range got {
			if wantMsg == gotMsg {
				found = true
				continue
			}
		}
		if !found {
			t.Errorf("'%s' not found on ES", wantMsg)
		}
	}
}

func timeBetweenLogEntries(t *testing.T, l1, l2 string) time.Duration {
	type entry struct {
		TS string `json:"@timestamp"`
	}

	e1 := entry{}
	if err := json.Unmarshal([]byte(l1), &e1); err != nil {
		t.Fatalf("cannot parse log entry. Err: %s. Entry: %s", err, l1)
	}

	e2 := entry{}
	if err := json.Unmarshal([]byte(l2), &e2); err != nil {
		t.Fatalf("cannot parse log entry. Err: %s. Entry: %s", err, l1)
	}

	t1, err := time.Parse("2006-01-02T15:04:05Z0700", e1.TS)
	if err != nil {
		t.Fatalf("cannot parse time from first log entry: %s", err)
	}

	t2, err := time.Parse("2006-01-02T15:04:05Z0700", e2.TS)
	if err != nil {
		t.Fatalf("cannot parse time from second log entry: %s", err)
	}

	return t2.Sub(t1)
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
		t.Fatalf("cannot stat file: %s", err)
	}

	return true
}

func waitForEOF(t *testing.T, filebeat *integration.BeatProc, files []string) {
	for _, path := range files {
		if runtime.GOOS == "windows" {
			path = strings.ReplaceAll(path, `\`, `\\`)
		}
		eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", path)

		require.Eventuallyf(
			t,
			func() bool {
				return filebeat.GetLogLine(eofMsg) != ""
			},
			5*time.Second,
			100*time.Millisecond,
			"EOF log not found for %q", path,
		)
	}
}

func waitForDidNotChange(t *testing.T, filebeat *integration.BeatProc, files []string) {
	for _, path := range files {
		eofMsg := fmt.Sprintf("File didn't change: %s", path)

		require.Eventuallyf(
			t,
			func() bool {
				return filebeat.GetLogLine(eofMsg) != ""
			},
			5*time.Second,
			100*time.Millisecond,
			"'File didn't change' log not found for %q", path,
		)
	}
}
