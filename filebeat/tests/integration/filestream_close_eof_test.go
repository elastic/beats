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

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestFilestreamCloseEOFReopenBeforeACKDoesNotDuplicateEvents(t *testing.T) {
	integration.EnsureESIsRunning(t)

	index := "test-close-eof-reopen-" + uuid.Must(uuid.NewV4()).String()
	esURL := integration.GetESAdminURL(t, "http")
	proxy, proxyURL := integration.NewDisablingProxy(t, esURL.String())

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()

	logFile := filepath.Join(workDir, "log.log")
	msgLogFilePath := logFile
	if runtime.GOOS == "windows" {
		msgLogFilePath = strings.ReplaceAll(logFile, `\`, `\\`)
	}

	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	cfgYAML := getConfig(t, map[string]any{
		"homePath": workDir,
		"logfile":  logFile,
		"esHost":   proxyURL,
		"user":     user,
		"pass":     pass,
		"index":    index,
	}, "", "filestream_close_eof_reopen_before_ack.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	proxy.Disable()

	firstBatch := logFileLines[:5]
	secondBatch := logFileLines[5:]
	writeLines(t, logFile, firstBatch, false)

	closeMsg := fmt.Sprintf("EOF has been reached. Closing. Path='%s'", msgLogFilePath)
	filebeat.WaitLogsContains("Starting harvester for file", 10*time.Second, "first harvester did not start")
	filebeat.WaitLogsContains(closeMsg, 10*time.Second, "first harvester did not close on EOF")
	filebeat.WaitLogsContains("Stopped harvester for file", 10*time.Second, "first harvester did not stop")

	writeLines(t, logFile, secondBatch, true)

	filebeat.WaitLogsContains("Starting harvester for file", 10*time.Second, "second harvester did not start")
	filebeat.WaitLogsContains(closeMsg, 10*time.Second, "second harvester did not close on EOF")
	filebeat.WaitLogsContains("Stopped harvester for file", 10*time.Second, "second harvester did not stop")

	proxy.Enable()

	// Wait for events to start being published
	filebeat.WaitLogsContains("events have been sent to elasticsearch in", 5*time.Second, "events were not published")

	// Wait a couple more seconds to ensure we flush the queue
	time.Sleep(2 * time.Second)
	filebeat.Stop()
	filebeat.WaitLogsContains("filebeat stopped.", 2*time.Second, "Filebeat did not stop successfully")

	got := integration.GetEventsMsgFromES(t, index, 200)
	if have, want := len(got), len(logFileLines); have != want {
		t.Errorf("the wrong number of events were published to ES, expecting %d, got %d", want, have)
		t.Log("Events published:")
		for i, e := range got {
			t.Logf("[%02d] '%s'", i, e)
		}
	}
}

func writeLines(t *testing.T, path string, lines []string, appendData bool) {
	t.Helper()

	flag := os.O_CREATE | os.O_WRONLY
	if appendData {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(path, flag, 0o644)
	require.NoError(t, err, "cannot open log file")
	defer f.Close()

	_, err = f.WriteString(strings.Join(lines, "\n") + "\n")
	require.NoError(t, err, "cannot write log file")
	require.NoError(t, f.Sync(), "cannot sync log file")
}
