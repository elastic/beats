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
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
)

// TestFilestreamFingerprintFileReplacement covers what the harvester reads when
// the fingerprint of its file shows up in a different file at another path.
// When several files share a fingerprint, the scanner reports the path that sorts first.
func TestFilestreamFingerprintFileReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("files open for reading cannot be removed or replaced on Windows")
	}

	const header = "header shared by all files, longer than the 64-byte fingerprint window"

	tests := []struct {
		name    string
		config  map[string]any // optional overrides of the base config
		actions []action
		want    []string // every published message, in order
	}{
		{
			name: "renaming the same file keeps reading it",
			actions: []action{
				write("a", header, "a-1"), scanCycle, ingested(header, "a-1"),
				rename("a", "b"), appendTo("b", "a-2"),
			},
			want: []string{header, "a-1", "a-2"},
		},
		{
			name: "a new file replaces a deleted one in the same scan",
			actions: []action{
				write("a", header, "a-1"), scanCycle, ingested(header, "a-1"),
				// Writing "b" first leaves no scan without the fingerprint.
				write("b", header, "b-1", "b-2"), remove("a"),
			},
			want: []string{header, "a-1", "b-2"},
		},
		{
			name: "a shorter new file is read once it grows past the stored offset",
			actions: []action{
				write("a", header, "a-1", "a-2"), scanCycle, ingested(header, "a-1", "a-2"),
				write("b", header, "b-1"), remove("a"), scanCycle,
				appendTo("b", "b-2", "b-3"),
			},
			want: []string{header, "a-1", "a-2", "b-3"},
		},
		{
			name:   "a harvester that is behind reads the old file to its end first",
			config: map[string]any{"backoff.init": "2s", "backoff.max": "2s", "close.on_state_change.check_interval": "2s"},
			actions: []action{
				write("a", header, "a-1"), scanCycle, ingested(header, "a-1"),
				// The harvester is parked, so it has not read a-2 when the scanner reports "b".
				appendTo("a", "a-2"), write("b", header, "b-1", "b-2", "b-3"), remove("a"),
			},
			want: []string{header, "a-1", "a-2", "b-3"},
		},
		{
			name: "a new file created before the old one is deleted",
			actions: []action{
				write("a", header, "a-1"), scanCycle, ingested(header, "a-1"),
				write("b", header, "b-1", "b-2"), scanCycle,
				remove("a"),
			},
			want: []string{header, "a-1", "b-2"},
		},
		{
			name: "a copy continues where the original stopped",
			actions: []action{
				write("a", header, "a-1"), scanCycle, ingested(header, "a-1"),
				copyContent("a", "b"), scanCycle,
				remove("a"), scanCycle,
				appendTo("b", "a-2"),
			},
			want: []string{header, "a-1", "a-2"},
		},
		{
			name:   "a compressed copy adds no duplicates",
			config: map[string]any{"compression": "auto"},
			actions: []action{
				write("a", header, "a-1", "a-2"), scanCycle, ingested(header, "a-1", "a-2"),
				gzipFile("a"), scanCycle,
			},
			want: []string{header, "a-1", "a-2"},
		},
		{
			name: "a copy that sorts first does not take over a live file",
			actions: []action{
				write("b", header, "b-1"), scanCycle, ingested(header, "b-1"),
				copyContent("b", "a"), scanCycle,
				appendTo("b", "b-2"),
			},
			want: []string{header, "b-1", "b-2"},
		},
		{
			name: "a longer file that sorts first takes over a live file",
			actions: []action{
				write("b", header, "b-1"), scanCycle, ingested(header, "b-1"),
				write("a", header, "a-1", "a-2"), scanCycle,
				// The old harvester has stopped before "b" grows, so it cannot read b-2.
				ingested(header, "b-1", "a-2"),
				appendTo("b", "b-2"), scanCycle,
			},
			want: []string{header, "b-1", "a-2"},
		},
		{
			// The scanner reports a delete and then a create. clean_removed removes
			// the state on the delete, so the new file is read from the start.
			name: "a new file that appears a scan after the old file is deleted",
			actions: []action{
				write("a", header, "a-1"), scanCycle, ingested(header, "a-1"),
				remove("a"), scanCycle,
				write("b", header, "b-1", "b-2"),
			},
			want: []string{header, "a-1", header, "b-1", "b-2"},
		},
		{
			// "short" is under the fingerprint length, so the prefix match reports
			// a rename and "b" is read after "short\n".
			name: "growing fingerprint: a new file that starts with a deleted short file's bytes",
			actions: []action{
				write("a", "short"), scanCycle, ingested("short"),
				// A scan before the append sees an exact rename, one after it a prefix rename.
				write("b", "short"), remove("a"), appendTo("b", header, "b-1"),
			},
			want: []string{"short", header, "b-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			startReplacementInput(t, env, tc.config)

			for i, a := range tc.actions {
				t.Logf("step %d: %s", i+1, a.desc)
				a.do(t, env)
			}

			ingested(tc.want...).do(t, env)
			scanCycle.do(t, env)
			assert.Equal(t, tc.want, env.getOutputMessages(), "published messages one scan after the last step")
		})
	}
}

// waitTimeout bounds every wait for a scan or for published messages.
const waitTimeout = 10 * time.Second

// action is one step of a scenario, applied to the files or to the running input.
type action struct {
	desc string
	do   func(t *testing.T, env *inputTestingEnvironment)
}

// write creates or overwrites name with lines, each followed by a newline.
func write(name string, lines ...string) action {
	return action{
		desc: fmt.Sprintf("write %s %q", name, lines),
		do: func(_ *testing.T, env *inputTestingEnvironment) {
			env.mustWriteToFile(name, joinLines(lines))
		},
	}
}

// appendTo appends lines to name, each followed by a newline.
func appendTo(name string, lines ...string) action {
	return action{
		desc: fmt.Sprintf("append %s %q", name, lines),
		do: func(_ *testing.T, env *inputTestingEnvironment) {
			env.mustAppendToFile(name, joinLines(lines))
		},
	}
}

// rename moves from to to, keeping the same file.
func rename(from, to string) action {
	return action{
		desc: fmt.Sprintf("rename %s to %s", from, to),
		do: func(_ *testing.T, env *inputTestingEnvironment) {
			env.mustRenameFile(from, to)
		},
	}
}

// remove deletes name.
func remove(name string) action {
	return action{
		desc: "remove " + name,
		do: func(_ *testing.T, env *inputTestingEnvironment) {
			env.mustRemoveFile(name)
		},
	}
}

// copyContent writes the content of from to a new file named to.
func copyContent(from, to string) action {
	return action{
		desc: fmt.Sprintf("copy %s to %s", from, to),
		do: func(t *testing.T, env *inputTestingEnvironment) {
			copyFile(t, env.abspath(from), env.abspath(to))
		},
	}
}

// gzipFile replaces name with a gzip-compressed copy named name+".gz".
func gzipFile(name string) action {
	return action{
		desc: "gzip " + name,
		do: func(t *testing.T, env *inputTestingEnvironment) {
			data, err := os.ReadFile(env.abspath(name))
			require.NoErrorf(t, err, "read %s", name)

			// Stage outside the watched directory and rename it in, so no scan
			// sees a partially written archive.
			staged := filepath.Join(t.TempDir(), name+".gz")
			//nolint:gosec // All paths are controlled by the test code.
			require.NoErrorf(t, os.WriteFile(staged, gziptest.Compress(t, data, gziptest.CorruptNone), 0o644), "write %s", staged)
			require.NoErrorf(t, os.Rename(staged, env.abspath(name+".gz")), "move %s in", staged)
			env.mustRemoveFile(name)
		},
	}
}

// scanCycle waits until the scanner has observed every earlier change to the files.
// The shared directory listing cache can serve a scan a listing up to one
// check_interval old, so it waits for three full scans that start after this point.
var scanCycle = action{
	desc: "scan cycle",
	do: func(t *testing.T, env *inputTestingEnvironment) {
		// FindInLogs advances past every line it does not match, and a NUL byte
		// is never logged, so this skips the scans that started earlier.
		_, err := env.testLogger.FindInLogs("\x00")
		require.NoError(t, err, "skip the logs written before the scan cycle")
		for i := range 3 {
			waitForLog(t, env, "Start next scan", "scan %d did not start", i+1)
			waitForLog(t, env, "File scan complete", "scan %d did not complete", i+1)
		}
	},
}

// waitForLog waits for the next log line containing s. It polls faster than
// WaitLogsContains, whose 100ms interval matches the scan interval.
func waitForLog(t *testing.T, env *inputTestingEnvironment, s string, msgAndArgs ...any) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		found, err := env.testLogger.FindInLogs(s)
		require.NoError(c, err, "read the test log")
		assert.True(c, found, "log line %q", s)
	}, waitTimeout, 25*time.Millisecond, msgAndArgs...)
}

// ingested waits until exactly lines have been published so far, in order.
func ingested(lines ...string) action {
	return action{
		desc: fmt.Sprintf("ingested %q", lines),
		do: func(t *testing.T, env *inputTestingEnvironment) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				assert.Equal(c, lines, env.getOutputMessages())
			}, waitTimeout, 10*time.Millisecond, "published messages")
		},
	}
}

func joinLines(lines []string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

// startReplacementInput runs an input with the shared configuration and
// overrides until the test ends.
func startReplacementInput(t *testing.T, env *inputTestingEnvironment, overrides map[string]any) {
	t.Helper()
	id := "fake-ID-" + uuid.Must(uuid.NewV4()).String()
	cfg := map[string]any{
		"id":                                    id,
		"paths":                                 []string{env.abspath("*")},
		"file_identity.fingerprint":             map[string]any{},
		"prospector.scanner.fingerprint.offset": 0,
		"prospector.scanner.fingerprint.length": 64,
		"prospector.scanner.check_interval":     "100ms",
		"backoff.init":                          "10ms",
		"backoff.max":                           "50ms",
	}
	maps.Copy(cfg, overrides)

	inp := env.mustCreateInput(cfg)
	ctx, cancelInput := context.WithCancel(t.Context())
	t.Cleanup(func() {
		cancelInput()
		env.waitUntilInputStops()
	})
	env.startInput(ctx, id, inp)
}
