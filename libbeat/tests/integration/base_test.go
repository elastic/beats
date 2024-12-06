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
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBase(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	mockbeat.Stop()
	mockbeat.WaitForLogs("mockbeat stopped.", 30*time.Second)
}

func TestSigHUP(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sighup not supported on windows")
	}
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	err := mockbeat.Process.Signal(syscall.SIGHUP)
	require.NoErrorf(t, err, "error sending SIGHUP to mockbeat")
	mockbeat.Stop()
	mockbeat.WaitForLogs("mockbeat stopped.", 30*time.Second)
}

func TestNoConfig(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("error loading config file", 10*time.Second)
}

func TestInvalidConfig(t *testing.T) {
	cfg := `
test:
  test were
  : invalid yml
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("error loading config file", 10*time.Second)
}

func TestInvalidCLI(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-d", "config", "-E", "output.console=invalid")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("error unpacking config data", 10*time.Second)
}

func TestConsoleOutput(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-e")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitStdErrContains("mockbeat start running.", 10*time.Second)
	mockbeat.WaitStdOutContains("Mockbeat is alive", 10*time.Second)
}

func TestConsoleBulkMaxSizeOutput(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
    bulk_max_size: 1
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-e")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitStdErrContains("mockbeat start running.", 10*time.Second)
	mockbeat.WaitStdOutContains("Mockbeat is alive", 10*time.Second)
}

func TestLoggingMetrics(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
logging:
  metrics:
    period: 0.1s
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-e")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitStdErrContains("mockbeat start running.", 10*time.Second)
	mockbeat.WaitStdErrContains("Non-zero metrics in the last", 10*time.Second)
	mockbeat.Stop()
	mockbeat.WaitStdErrContains("Total metrics", 10*time.Second)
}

func TestPersistentUuid(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-e")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitStdErrContains("mockbeat start running.", 10*time.Second)

	metaFile1, err := mockbeat.LoadMeta()
	require.NoError(t, err, "error opening meta.json file")
	mockbeat.Stop()
	mockbeat.WaitStdErrContains("Beat ID: "+metaFile1.UUID.String(), 10*time.Second)
	mockbeat.Start()
	mockbeat.WaitStdErrContains("mockbeat start running.", 10*time.Second)
	metaFile2, err := mockbeat.LoadMeta()
	require.Equal(t, metaFile1.UUID.String(), metaFile2.UUID.String())
}
