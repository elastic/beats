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
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testingintegration "github.com/elastic/beats/v7/filebeat/testing/integration"
)

func TestWaitUntilEOF(t *testing.T) {
	prefix := strings.Repeat("a", 1024)
	events := 10
	logGen := testingintegration.NewJSONGenerator(prefix)
	_, files := testingintegration.GenerateLogFiles(t,
		1, events, logGen)
	path := files[0]

	env := newInputTestingEnvironment(t)
	id := "TestWaitUntilEOF"
	waitEOFTimeout := 30 * time.Second
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                     id,
		"paths":                  []string{path},
		"read_until_eof.enabled": true,
		"read_until_eof.timeout": waitEOFTimeout,
		"close.reader.on_eof":    true,
	})

	env.pipeline.SetAllowedEvents(1)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.WaitLogsContains(fmt.Sprintf("A new file %s has been found", files[0]),
		time.Minute, 100*time.Millisecond)
	env.WaitLogsContains("Starting harvester for file",
		time.Minute, 100*time.Millisecond)

	cancelInput()
	env.pipeline.UnblockClients()

	env.WaitLogsContains(fmt.Sprintf("input closing, read_until_eof enabled, waiting EOF or %s timeout, whichever happens first", waitEOFTimeout),
		5*time.Second, 100*time.Millisecond)

	env.waitUntilEventCount(events)
	t.Log("after waitUntilEventCount 2nd batch")

	env.WaitLogsContains("read_until_eof enabled, EOF reached. closing input",
		waitEOFTimeout, 100*time.Millisecond)
}

// TestWaitUntilEOF_reachesEOFWithCloseOnEOF proves that
// `close.reader.on_eof: true` and `read_until_eof.enabled: true` coexist
// when no cancellation happens: the reader reads the file, reaches EOF,
// and closes via the close.reader.on_eof path. The readUntilEOF drain is
// not entered — its flag is harmless in this case, a guarantee the read until
// EOF loop only activates on input cancel.
// All events must be published, and the "input closing, read_until_eof
// enabled..." log must NOT appear, proving the readUntilEOF block is
// skipped when the reader reaches EOF on its own.
func TestWaitUntilEOF_reachesEOFWithCloseOnEOF(t *testing.T) {
	prefix := strings.Repeat("a", 1024)
	events := 10
	logGen := testingintegration.NewJSONGenerator(prefix)
	_, files := testingintegration.GenerateLogFiles(t,
		1, events, logGen)
	path := files[0]

	env := newInputTestingEnvironment(t)
	id := "TestWaitUntilEOF_reachesEOFWithCloseOnEOF"
	waitEOFTimeout := 30 * time.Second
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                     id,
		"paths":                  []string{path},
		"read_until_eof.enabled": true,
		"read_until_eof.timeout": waitEOFTimeout,
		"close.reader.on_eof":    true,
	})

	env.pipeline.SetAllowedEvents(events)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)
	env.startInput(ctx, id, inp)

	env.WaitLogsContains(fmt.Sprintf("A new file %s has been found", files[0]),
		time.Minute, 100*time.Millisecond)
	env.WaitLogsContains("Starting harvester for file",
		time.Minute, 100*time.Millisecond)

	env.waitUntilEventCount(events)

	// Wait for the harvester to actually close.
	env.WaitLogsContains("EOF has been reached. Closing.",
		10*time.Second, 100*time.Millisecond)

	// Prove the readUntilEOF block was skipped.
	found, err := env.testLogger.FindInLogs(
		"input closing, read_until_eof enabled, waiting EOF or")
	require.NoError(t, err, "could not scan log file")
	assert.Falsef(t, found,
		"readUntilEOF drain must not be entered when the reader reaches "+
			"EOF naturally via close.reader.on_eof")
}

// TestWaitUntilEOF_gzipFile exercises read_until_eof with a
// gzip-compressed log file and pipeline backpressure. The harvester is
// blocked on the first publish when the input is cancelled; the
// remaining events must still be read out of the gzip stream and
// published via the readUntilEOF mode.
func TestWaitUntilEOF_gzipFile(t *testing.T) {
	prefix := strings.Repeat("a", 1024)
	events := 10
	logGen := testingintegration.NewJSONGenerator(prefix)
	_, files := testingintegration.GenerateGZIPLogFiles(t, 1, events, logGen)
	path := files[0]

	env := newInputTestingEnvironment(t)
	id := "TestWaitUntilEOF_gzipFile"
	waitEOFTimeout := 30 * time.Second
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                     id,
		"paths":                  []string{path},
		"compression":            "auto",
		"read_until_eof.enabled": true,
		"read_until_eof.timeout": waitEOFTimeout,
	})

	env.pipeline.SetAllowedEvents(1)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)
	env.startInput(ctx, id, inp)

	env.WaitLogsContains(fmt.Sprintf("A new file %s has been found", files[0]),
		time.Minute, 1*time.Second)
	env.WaitLogsContains("Starting harvester for file",
		time.Minute, 1*time.Second)

	cancelInput()
	env.pipeline.UnblockClients()

	env.WaitLogsContains(fmt.Sprintf(
		"input closing, read_until_eof enabled, waiting EOF or %s timeout, whichever happens first",
		waitEOFTimeout),
		5*time.Second, 1*time.Second)

	env.waitUntilEventCount(events)
	env.WaitLogsContains("read_until_eof enabled, EOF reached. closing input",
		waitEOFTimeout, 1*time.Second)
}

// TestWaitUntilEOF_fileDeletedDuringReadUntilEOF deletes the file after
// readUntilEOF mode has started, and asserts the reader continues to drain
// the fd to EOF — the Linux fd-still-readable semantic. It runs with
// close.on_state_change.removed=true to also exercise 'periodicStateCheck'
// must have been stopped by startReadUntilEOF before the deletion, otherwise it
// would cancel the swapped-in readerCtx when it next ticked and the readUntilEOF
// mode would be aborted before reading the remaining bytes.
//
// Restricted to non-Windows because Windows file-deletion semantics
// (sharing modes, delete-on-close) differ.
func TestWaitUntilEOF_fileDeletedDuringReadUntilEOF(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux/macOS-only: relies on the open-fd-survives-unlink semantic")
	}
	prefix := strings.Repeat("a", 1024)
	events := 10
	logGen := testingintegration.NewJSONGenerator(prefix)
	_, files := testingintegration.GenerateLogFiles(t,
		1, events, logGen)
	path := files[0]

	env := newInputTestingEnvironment(t)
	id := "TestWaitUntilEOF_fileDeletedDuringReadUntilEOF"
	waitEOFTimeout := 30 * time.Second
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                            id,
		"paths":                         []string{path},
		"read_until_eof.enabled":        true,
		"read_until_eof.timeout":        waitEOFTimeout,
		"close.on_state_change.removed": true,
	})

	env.pipeline.SetAllowedEvents(1)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)
	env.startInput(ctx, id, inp)

	env.WaitLogsContains(fmt.Sprintf("A new file %s has been found", files[0]),
		time.Minute, 100*time.Millisecond)
	env.WaitLogsContains("Starting harvester for file",
		time.Minute, 100*time.Millisecond)

	cancelInput()
	// Wait for the input cancellation to propagate down to the harvester
	// (input ctx → managedInput.cancelCtx → prospector ctx → prospector
	// exits → stopHarvesterGroup → tg.Stop → tg.ctx → harvesterCtx).
	// Without this wait, the reader can wake from the blocked publish
	// before harvesterCtx is cancelled, see the loop condition still
	// satisfied, iterate, and block on the next publish — never entering
	// the readUntilEOF block.
	time.Sleep(1 * time.Second)
	// Allow exactly one more event so the publish blocked at cancel time
	// can return; handleReadError requires publish to
	// return before the outer loop re-checks ctx.Cancelation and exits
	// into the readUntilEOF block. The next read inside readUntilEOF
	// then blocks on the *next* publish, which is exactly the state we
	// want before we delete the file.
	env.pipeline.AllowMoreEvents(1)

	env.WaitLogsContains(fmt.Sprintf(
		"input closing, read_until_eof enabled, waiting EOF or %s timeout, whichever happens first",
		waitEOFTimeout),
		5*time.Second, 100*time.Millisecond)

	// Delete the file while the reader is still inside the readUntilEOF
	// loop, blocked on a publish. On Linux the open fd remains valid; the
	// reader must continue draining until EOF once we release the pipeline.
	require.NoError(t, os.Remove(path))

	// Release backpressure: the reader publishes the queued event and
	// keeps reading from the deleted-but-still-open fd until EOF.
	env.pipeline.UnblockClients()

	// All events still make it through, and readUntilEOF exits via the
	// EOF path (not via timeout-reached).
	env.waitUntilEventCount(events)
	env.WaitLogsContains("read_until_eof enabled, EOF reached. closing input",
		waitEOFTimeout, 100*time.Millisecond)
}

func TestWaitUntilEOF_timeout(t *testing.T) {
	prefix := strings.Repeat("a", 1024)
	events := 10
	wantEvents := 0
	logGen := testingintegration.NewJSONGenerator(prefix)
	_, files := testingintegration.GenerateLogFiles(t,
		1, events, logGen)
	path := files[0]

	env := newInputTestingEnvironment(t)
	id := "TestWaitUntilEOF"
	waitEOFTimeout := 1 * time.Second
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                     id,
		"paths":                  []string{path},
		"read_until_eof.enabled": true,
		"read_until_eof.timeout": waitEOFTimeout,
		"close.reader.on_eof":    true,
	})

	env.pipeline.SetAllowedEvents(1)
	wantEvents++

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.WaitLogsContains(fmt.Sprintf("A new file %s has been found", files[0]),
		time.Second, "no file found to be ingested")
	env.WaitLogsContains("Starting harvester for file",
		time.Second, "harvester did not start")

	cancelInput()

	// The input's ctx.Cancelation reaches the harvester's ctx.Cancelation only
	// after it has travelled through the prospector-exit / stopHarvesterGroup /
	// tg.Stop chain, which crosses several goroutine-scheduling boundaries.
	// Without this time.Sleep, the harvester can unblock from AllowMoreEvents
	// below, publish event 2, and race past the for-loop condition check before
	// observing the cancel — landing it back in waitIfBlocked on
	// event 3 with no path out until the WaitLogsContains times out.
	time.Sleep(500 * time.Millisecond)
	env.pipeline.AllowMoreEvents(1)
	wantEvents++

	env.WaitLogsContains(fmt.Sprintf("input closing, read_until_eof enabled, waiting EOF or %s timeout, whichever happens first", waitEOFTimeout),
		1*time.Second, "readUntilEOF starting log not found")

	// Wait the readUntilEOF timeout to expire, then unblock the pipeline so
	// `readLineFromSource` unblocks and the timeout branch is executed.
	time.Sleep(waitEOFTimeout + 500*time.Millisecond)
	env.pipeline.UnblockClients()
	wantEvents++ // only one more event should be published

	env.WaitLogsContains(fmt.Sprintf(
		"read_until_eof enabled, %s timeout reached. closing input", waitEOFTimeout),
		waitEOFTimeout+30*time.Second, "timeout log not found")

	assert.Len(t, env.pipeline.GetAllEvents(), wantEvents,
		"unexpected number of events published")
}
