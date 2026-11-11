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

package filestream

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/pkg/monitoring"
)

// keptOpenWindow is how long a kept-open file is watched for an unexpected
// close: 10 close.on_state_change.check_interval polls of the parked file.
const keptOpenWindow = 500 * time.Millisecond

// highCardinalityInputConfig returns the ID and the config of an input running
// the high_cardinality preset over glob. Scans and state checks run every
// 50ms so a scenario takes a fraction of a second per step. The inactivity
// timeout is long so that only EOF can close a file unless a test lowers it.
func highCardinalityInputConfig(env *inputTestingEnvironment, glob string) (string, map[string]any) {
	id := "high-cardinality-" + uuid.Must(uuid.NewV4()).String()
	return id, map[string]any{
		"id":                                     id,
		"paths":                                  []string{env.abspath(glob)},
		"preset.high_cardinality":                nil,
		"prospector.scanner.check_interval":      "50ms",
		"close.on_state_change.check_interval":   "50ms",
		"close.on_state_change.inactive":         "1h",
		"prospector.scanner.fingerprint.enabled": false,
		"file_identity.native":                   map[string]any{},
	}
}

// mustCreateHighCardinalityInput creates the input and fails the test if the
// preset did not resolve to high_cardinality, so no scenario can pass on the
// default preset.
func mustCreateHighCardinalityInput(t *testing.T, env *inputTestingEnvironment, cfg map[string]any) v2.Input {
	t.Helper()
	inp := env.mustCreateInput(cfg)
	env.logContains("preset resolved: high_cardinality;")
	return inp
}

// fileCounts are the per-input file counters: opened and closed totals and
// the files open right now.
type fileCounts struct {
	opened, closed, active uint64
}

// readFileCounts returns the file counters of input id, and false while its
// metrics registry does not exist.
func readFileCounts(env *inputTestingEnvironment, id string) (fileCounts, bool) {
	reg, ok := env.monitoring.InputsRegistry().Get(id).(*monitoring.Registry)
	if !ok {
		return fileCounts{}, false
	}
	get := func(name string) uint64 {
		v, _ := reg.Get(name).(*monitoring.Uint)
		if v == nil {
			return 0
		}
		return v.Get()
	}
	return fileCounts{
		opened: get("files_opened_total"),
		closed: get("files_closed_total"),
		active: get("files_active"),
	}, true
}

func waitFileCounts(t *testing.T, env *inputTestingEnvironment, id string, want fileCounts, msg string) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		got, ok := readFileCounts(env, id)
		assert.True(c, ok, "the metrics registry of input %q must exist", id)
		assert.Equal(c, want, got, "unexpected file counters")
	}, 10*time.Second, 10*time.Millisecond, msg)
}

// requireKeptOpen asserts that the harvester re-opened for path keeps it open
// at EOF. want is the counters with the file open. The 'Backoff now' line is
// only logged when the reader is not closed at EOF, which proves the decision;
// the window then proves no poll of the parked file closes it.
func requireKeptOpen(t *testing.T, env *inputTestingEnvironment, id, path string, want fileCounts) {
	t.Helper()
	env.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logPath(path)),
		10*time.Second,
		"a file that grew after an EOF close must be kept open at EOF")
	waitFileCounts(t, env, id, want, "the file must be open after it was read to EOF")
	require.Never(t, func() bool {
		got, _ := readFileCounts(env, id)
		return got != want
	}, keptOpenWindow, 10*time.Millisecond, "a kept-open file must stay open while it is active")
}

// messages returns the message of every published event, in publishing order
// per harvester.
func messages(env *inputTestingEnvironment) []string {
	var msgs []string
	for _, evt := range env.pipeline.GetAllEvents() {
		msg, _ := evt.Fields["message"].(string)
		msgs = append(msgs, msg)
	}
	return msgs
}

// logPath returns path as it appears in the JSON log file.
func logPath(path string) string {
	return strings.ReplaceAll(path, `\`, `\\`)
}

func TestHighCardinalityRestartSkipsIngestedFiles(t *testing.T) {
	env := newInputTestingEnvironment(t)
	id, cfg := highCardinalityInputConfig(env, "*.log")
	inp := mustCreateHighCardinalityInput(t, env, cfg)

	names := []string{"a.log", "b.log", "c.log"}
	sizes := map[string]int{}
	for _, name := range names {
		content := []byte(name + " line 1\n" + name + " line 2\n")
		env.mustWriteToFile(name, content)
		sizes[name] = len(content)
	}

	ctx, cancel := context.WithCancel(t.Context())
	env.startInput(ctx, id, inp)
	env.waitUntilEventCount(2 * len(names))
	for _, name := range names {
		env.requireOffsetInRegistry(name, id, sizes[name])
	}
	waitFileCounts(t, env, id, fileCounts{opened: 3, closed: 3, active: 0},
		"every file must be closed at EOF before the restart")
	cancel()
	env.restartManager()

	inp = mustCreateHighCardinalityInput(t, env, cfg)
	ctx, cancel = context.WithCancel(t.Context())
	env.startInput(ctx, id, inp)

	// The decision for each file is logged when the first scan finds it, in
	// map order, so each line is searched from the start of the log. The
	// first input never logs it: its files only grew before they were found.
	for _, name := range names {
		env.testLogger.ResetOffset()
		env.WaitLogsContains(
			fmt.Sprintf("File %s is already read, not starting a harvester: offset %d, size %d",
				logPath(env.abspath(name)), sizes[name], sizes[name]),
			10*time.Second, "a fully ingested file must be skipped after a restart")
	}
	// A harvester starts asynchronously after the skip decision would have
	// started it; the window is far longer than opening and reading a file
	// and covers 10 more scans.
	assert.Never(t, func() bool {
		got, _ := readFileCounts(env, id)
		return got.opened > 0 || len(env.pipeline.GetAllEvents()) > 2*len(names)
	}, 500*time.Millisecond, 10*time.Millisecond, "no fully ingested file must be opened after a restart")

	cancel()
	env.waitUntilInputStops()
	for _, name := range names {
		env.requireOffsetInRegistry(name, id, sizes[name])
	}
}

func TestHighCardinalityLifecycle(t *testing.T) {
	env := newInputTestingEnvironment(t)
	id, cfg := highCardinalityInputConfig(env, "*.log")
	// The kept-open window starts within milliseconds of the last read and
	// lasts keptOpenWindow; 3s leaves over 2s of scheduling slack before the
	// inactivity close can happen, and bounds the test's waiting for it.
	cfg["close.on_state_change.inactive"] = "3s"
	inp := mustCreateHighCardinalityInput(t, env, cfg)

	path := env.mustWriteToFile("a.log", []byte("line 1\n"))
	eofClose := fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPath(path))

	ctx, cancel := context.WithCancel(t.Context())
	env.startInput(ctx, id, inp)
	// harvester_limit is not configured, so the runner must get the preset's.
	env.WaitLogsContains(
		fmt.Sprintf("starting filestream harvester (max %d open files)", highCardinalityHarvesterLimit),
		10*time.Second, "the preset's harvester limit must reach the harvester runner")

	env.waitUntilEventCount(1)
	env.WaitLogsContains(eofClose, 10*time.Second, "a new file must be closed at EOF")
	waitFileCounts(t, env, id, fileCounts{opened: 1, closed: 1, active: 0},
		"a new file must be closed at EOF")

	env.mustAppendToFile("a.log", []byte("line 2\n"))
	env.waitUntilEventCount(2)
	requireKeptOpen(t, env, id, path, fileCounts{opened: 2, closed: 1, active: 1})

	env.WaitLogsContains(fmt.Sprintf("File is inactive. Closing. Path='%s'", logPath(path)),
		10*time.Second, "the kept-open file must be closed on inactivity")
	waitFileCounts(t, env, id, fileCounts{opened: 2, closed: 2, active: 0},
		"the kept-open file must be closed on inactivity")

	env.mustAppendToFile("a.log", []byte("line 3\n"))
	env.waitUntilEventCount(3)
	env.WaitLogsContains(eofClose, 10*time.Second,
		"a file closed on inactivity must be closed at EOF when it grows")
	waitFileCounts(t, env, id, fileCounts{opened: 3, closed: 3, active: 0},
		"a file closed on inactivity must be closed at EOF when it grows")
	assert.Equal(t, []string{"line 1", "line 2", "line 3"}, messages(env),
		"every line must be published once")

	cancel()
	env.waitUntilInputStops()
}

// TestHighCardinalityWarnsAboutCloseOnEOF is the only test that reaches the
// warning through a real configuration.
func TestHighCardinalityWarnsAboutCloseOnEOF(t *testing.T) {
	env := newInputTestingEnvironment(t)
	_, cfg := highCardinalityInputConfig(env, "*.log")
	cfg["close.reader.on_eof"] = true
	mustCreateHighCardinalityInput(t, env, cfg)
	env.logContains("'close.reader.on_eof' is ignored: the high_cardinality preset decides per file whether it is closed at EOF")

	env.waitUntilInputStops()
}
