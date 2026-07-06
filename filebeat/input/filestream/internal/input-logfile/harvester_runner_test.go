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

//nolint:errcheck // It's a test file
package input_logfile

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	eventuallyTimeout  = 5 * time.Second
	eventuallyInterval = 10 * time.Millisecond
)

func requireEventually(t *testing.T, condition func() bool, msg string) {
	t.Helper()
	require.Eventually(t, condition, eventuallyTimeout, eventuallyInterval, msg)
}

var (
	errHarvester       = fmt.Errorf("harvester error")
	errPipelineConnect = fmt.Errorf("pipeline connect error")
)

// --- Tests --------------------------------------------------------------

// TestHarvesterRunner_StartReadsToEOFAndTearsDown asserts a started source is
// read once (a single ReadSlice returning SliceDone) and then fully torn down:
// removed from the runner's bookkeeping and its session closed.
func TestHarvesterRunner_StartReadsToEOFAndTearsDown(t *testing.T) {
	h := &fakeHarvester{} // default session: ReadSlice -> SliceDone
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"source should be read once and removed from bookkeeping")

	require.Equal(t, 1, h.opens(), "exactly one session should have been opened")
	sess := h.lastSession()
	require.Equal(t, 1, sess.readCount(), "the source should be read exactly once")
	require.True(t, sess.isClosed(), "the session should be closed after teardown")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_HarvesterLimitQueuesAndPromotes asserts harvester_limit is
// a hard cap on open files: no more than `limit` sources are open at once, the
// rest are queued in statusWaiting (registered, holding no fd and no goroutine),
// and queued sources are promoted as slots free until every source is harvested
// and torn down.
func TestHarvesterRunner_HarvesterLimitQueuesAndPromotes(t *testing.T) {
	const limit = 2
	const total = 5

	release := make(chan struct{})
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) {
			<-release // hold the slot until the test releases it
			return SliceDone, nil
		},
	}
	g := testHarvesterRunner(t, h, limit)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	srcs := make([]*testSource, total)
	for i := range total {
		srcs[i] = &testSource{name: fmt.Sprintf("/path/to/test/%d", i)}
		g.Start(startContext(t), srcs[i])
	}

	// At most `limit` sessions are open at once; the rest are queued.
	requireEventually(t, func() bool { return h.opens() == limit },
		"exactly the limit number of harvesters should open")
	requireEventually(t, func() bool { return g.countWaiting() == total-limit },
		"the remaining sources should be queued")
	assert.Never(t, func() bool { return h.opens() > limit }, 200*time.Millisecond, eventuallyInterval,
		"never more than the limit may be open at once")

	// Each queued source is registered but parked in statusWaiting: no fd, no
	// reader, just waiting for an open slot.
	waiting := 0
	for _, src := range srcs {
		if s, ok := g.statusOf(g.identifier.ID(src)); ok && s == statusWaiting {
			waiting++
		}
	}
	require.Equal(t, total-limit, waiting, "queued sources should be in statusWaiting")

	// Release everything: queued sources are promoted as slots free until all are
	// harvested and torn down.
	close(release)
	requireEventually(t, func() bool {
		return h.opens() == total && g.countStates() == 0
	}, "all sources should eventually be harvested and torn down")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_StopHarvestersTearsDownQueued asserts a shutdown tears
// down queued (never-opened) sources too, without leaking goroutines.
func TestHarvesterRunner_StopHarvestersTearsDownQueued(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 1)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src1 := &testSource{name: "/path/to/test/1"}
	src2 := &testSource{name: "/path/to/test/2"}
	id1 := g.identifier.ID(src1)
	id2 := g.identifier.ID(src2)
	g.Start(startContext(t), src1)
	g.Start(startContext(t), src2)

	// src1 takes the slot and blocks reading; src2 is queued.
	requireEventually(t, func() bool {
		s, ok := g.statusOf(id2)
		return h.opens() == 1 && ok && s == statusWaiting
	}, "one source open, the other queued")

	require.NoError(t, g.StopHarvesters())
	require.False(t, g.hasID(id1), "running source should be torn down")
	require.False(t, g.hasID(id2), "queued source should be torn down")
}

// TestHarvesterRunner_StopCancelsRunningHarvester asserts Stop cancels an
// in-progress read and the source is torn down.
func TestHarvesterRunner_StopCancelsRunningHarvester(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	// Wait until the session is actually open (status becomes Running before
	// setup runs, so gate on opens() to guarantee a read is in progress).
	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusRunning && h.opens() == 1
	}, "harvester should be running with an open session before Stop")

	g.Stop(src)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"source should be removed after Stop")
	require.True(t, h.lastSession().isClosed(), "session should be closed after Stop")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_SameSourceStartedOnce asserts that repeated Start calls for
// a source that is already being harvested are ignored (no second session is
// opened), regardless of the harvester limit.
func TestHarvesterRunner_SameSourceStartedOnce(t *testing.T) {
	for _, limit := range []uint64{0, 1, 100} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			h := &fakeHarvester{readFn: blockUntilCancelled}
			g := testHarvesterRunner(t, h, limit)

			goroutines := resources.NewGoroutinesChecker()
			defer goroutines.WaitUntilOriginalCount()

			g.start()
			src := &testSource{name: "/path/to/test"}
			id := g.identifier.ID(src)
			g.Start(startContext(t), src)

			requireEventually(t, func() bool { return h.opens() == 1 && g.hasID(id) },
				"first harvester should be registered and running")

			// Hammer Start for the same source; none should open a new session.
			for range 20 {
				g.Start(startContext(t), src)
			}
			assert.Never(t, func() bool { return h.opens() > 1 }, 200*time.Millisecond, eventuallyInterval,
				"no additional harvester should start for an already-running source")

			g.Stop(src)
			require.NoError(t, g.StopHarvesters())
			require.Equal(t, 1, h.opens())
		})
	}
}

// TestHarvesterRunner_Restart asserts Restart stops the running harvester and
// starts a fresh one (a second session is opened) for the same source.
func TestHarvesterRunner_Restart(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return h.opens() == 1 }, "first harvester should run")

	g.Restart(startContext(t), src)

	requireEventually(t, func() bool { return h.opens() == 2 && g.hasID(id) },
		"a second harvester should be started by Restart")
	require.True(t, h.session(0).isClosed(), "the first session should be closed by Restart")

	g.Stop(src)
	require.NoError(t, g.StopHarvesters())
	require.Equal(t, 2, h.opens())
}

// TestHarvesterRunner_ReadErrorTearsDown asserts a source whose read returns an
// error is torn down (removed and its session closed) rather than retried.
func TestHarvesterRunner_ReadErrorTearsDown(t *testing.T) {
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) {
			return SliceDone, errHarvester
		},
	}
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"source should be removed after a read error")
	require.Equal(t, 1, h.opens())
	require.True(t, h.lastSession().isClosed(), "session should be closed after a read error")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_OpenSessionErrorTearsDown asserts that when OpenSession
// fails during setup, the acquired client and resource lock are released and the
// source is torn down.
func TestHarvesterRunner_OpenSessionErrorTearsDown(t *testing.T) {
	h := &fakeHarvester{openErr: errHarvester}
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"source should be removed when OpenSession fails")
	require.Equal(t, 0, h.opens(), "no session is produced when OpenSession errors")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_ConnectErrorTearsDown asserts that when the pipeline fails
// to connect during setup, the source is torn down and no session is opened.
func TestHarvesterRunner_ConnectErrorTearsDown(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)
	g.pipeline = &MockPipeline{connectErr: errPipelineConnect}

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"source should be removed when the pipeline connect fails")
	require.Equal(t, 0, h.opens(), "no session should be opened when ConnectWith fails")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_ConnectErrorDegradesStatus asserts that a pipeline connect
// failure during setup is a permanent harvester error and degrades the input's
// status (rather than being silently retried).
func TestHarvesterRunner_ConnectErrorDegradesStatus(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)
	g.pipeline = &MockPipeline{connectErr: errPipelineConnect}

	g.start()
	rec := &recordingStatusReporter{}
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t).WithStatusReporter(rec), src)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"source should be removed when the pipeline connect fails")
	requireEventually(t, func() bool { return rec.last() == status.Degraded },
		"a permanent setup failure must degrade the input status")
	require.Contains(t, rec.lastMsg(), "test-input",
		"the degraded message should identify the input")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_GZIPLifecycleMetrics asserts the GZIP-specific lifecycle
// gauges/counters are incremented when a GZIP source opens and balanced when it
// is torn down.
func TestHarvesterRunner_GZIPLifecycleMetrics(t *testing.T) {
	h := &fakeHarvester{gzip: true, readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)
	m := g.metrics

	g.start()
	src := &testSource{name: "/path/to/test.gz"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	// setup() runs in the reader goroutine, so wait until it has incremented the
	// GZIP open metrics before asserting the rest.
	requireEventually(t, func() bool { return m.HarvesterGZIPStarted.Get() == 1 },
		"GZIP source should be opened")

	require.Equal(t, uint64(1), m.FilesGZIPActive.Get(), "gzip_files_active")
	require.Equal(t, int64(1), m.HarvesterGZIPRunning.Get(), "harvester gzip_running")
	require.Equal(t, uint64(1), m.FilesGZIPOpened.Get(), "gzip_files_opened_total")
	require.Equal(t, int64(1), m.HarvesterOpenGZIPFiles.Get(), "harvester gzip_open_files")
	require.Equal(t, int64(1), m.HarvesterGZIPStarted.Get(), "harvester gzip_started")

	g.Stop(src)
	requireEventually(t, func() bool { return !g.hasID(id) },
		"GZIP source should be torn down after Stop")

	require.Equal(t, uint64(0), m.FilesGZIPActive.Get(), "gzip_files_active must return to zero")
	require.Equal(t, int64(0), m.HarvesterGZIPRunning.Get(), "harvester gzip_running must return to zero")
	require.Equal(t, int64(0), m.HarvesterOpenGZIPFiles.Get(), "harvester gzip_open_files must return to zero")
	require.Equal(t, uint64(1), m.FilesGZIPClosed.Get(), "gzip_files_closed_total")
	require.Equal(t, int64(1), m.HarvesterGZIPClosed.Get(), "harvester gzip_closed")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_ParkAndResume drives the park/resume lifecycle directly
// (without the time-based waker): a read that yields parks the source, a poll
// that reports new data resumes it, and the resumed read reaching EOF tears it
// down.
func TestHarvesterRunner_ParkAndResume(t *testing.T) {
	h := &fakeHarvester{
		readFn: func(call int, _ v2.Context) (SliceVerdict, error) {
			if call == 1 {
				return SliceYield, nil // caught up to EOF: park
			}
			return SliceDone, nil // resumed read reaches EOF
		},
		pollFn: func(_ int) PollResult { return PollResume }, // new data available
	}
	g := testHarvesterRunner(t, h, 0)
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	// The first read yields, so the source parks.
	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked
	}, "source should park after a yielding read")
	active, parked := g.counts()
	require.Equal(t, 0, active)
	require.Equal(t, 1, parked, "parked gauge should count the parked source")

	// Simulate the waker firing: pop the due source and poll it.
	state := g.popDueNow()
	require.NotNil(t, state, "the parked source should be due")
	g.pollParked(state)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"resumed source should be read again and torn down at EOF")
	sess := h.lastSession()
	require.Equal(t, 2, sess.readCount(), "source should be read twice (initial + resume)")
	require.Equal(t, 1, sess.pollCount())
	require.True(t, sess.isClosed())

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_ParkThenClose asserts a poll reporting a close condition
// tears a parked source down.
func TestHarvesterRunner_ParkThenClose(t *testing.T) {
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) { return SliceYield, nil },
		pollFn: func(_ int) PollResult { return PollClose },
	}
	g := testHarvesterRunner(t, h, 0)
	src := &testSource{name: "/path/to/test"}
	id, state := startParkedAndClaimDue(t, g, src)
	g.pollParked(state)

	requireEventually(t, func() bool { return !g.hasID(id) },
		"a parked source should tear down on a close poll")
	require.True(t, h.lastSession().isClosed())

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_ParkPollGrowsBackoff asserts that a still-idle poll
// (PollPark) re-parks the source with a grown backoff.
func TestHarvesterRunner_ParkPollGrowsBackoff(t *testing.T) {
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) { return SliceYield, nil },
		pollFn: func(_ int) PollResult { return PollPark },
	}
	g := testHarvesterRunner(t, h, 0)
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked
	}, "source should park")

	state, ok := g.stateFor(id)
	require.True(t, ok)
	require.Equal(t, minWakerBackoff, state.backoff, "a progressing read parks with the minimum backoff")

	state = g.popDueNow()
	require.NotNil(t, state)
	g.pollParked(state)

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked && state.backoff == growBackoff(minWakerBackoff)
	}, "an idle poll should re-park with a grown backoff")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_StopHarvestersStopsParked asserts StopHarvesters tears down
// a parked source and stops the waker (no goroutine leak).
func TestHarvesterRunner_StopHarvestersStopsParked(t *testing.T) {
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) { return SliceYield, nil },
		pollFn: func(_ int) PollResult { return PollPark },
	}
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start() // run the real waker
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return g.parkedLen() == 1 },
		"source should be parked and watched by the waker")

	require.NoError(t, g.StopHarvesters())
	require.False(t, g.hasID(id), "parked source should be torn down by StopHarvesters")
	require.True(t, h.lastSession().isClosed(), "parked session should be closed by StopHarvesters")
}

// TestHarvesterRunner_Continue asserts Continue carries a source over to a new
// identity and harvests the new source.
func TestHarvesterRunner_Continue(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	prev := &testSource{name: "/path/to/old"}
	next := &testSource{name: "/path/to/new"}
	nextID := g.identifier.ID(next)

	g.Continue(startContext(t), prev, next)

	requireEventually(t, func() bool { return h.opens() == 1 && g.hasID(nextID) },
		"Continue should start a harvester for the next source")
	require.False(t, g.hasID(g.identifier.ID(prev)), "the previous source should not be harvested")

	g.Stop(next)
	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_StopUnknownSourceIsNoop asserts Stop on a source that is
// not being harvested does nothing and does not panic.
func TestHarvesterRunner_StopUnknownSourceIsNoop(t *testing.T) {
	g := testHarvesterRunner(t, &fakeHarvester{}, 0)
	g.start()
	g.Stop(&testSource{name: "/never/started"}) // must be a no-op
	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_StopHarvestersIdempotent asserts StopHarvesters can be
// called more than once.
func TestHarvesterRunner_StopHarvestersIdempotent(t *testing.T) {
	g := testHarvesterRunner(t, &fakeHarvester{}, 0)
	g.start()
	require.NoError(t, g.StopHarvesters())
	require.NoError(t, g.StopHarvesters(), "StopHarvesters must be idempotent")
}

// TestHarvesterRunner_StartAfterShutdownIsIgnored asserts that once the runner is
// stopped, Start/Restart no longer create harvesters.
func TestHarvesterRunner_StartAfterShutdownIsIgnored(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 0)
	g.start()
	require.NoError(t, g.StopHarvesters())

	src := &testSource{name: "/path/to/test"}
	g.Start(startContext(t), src)   // enqueue sees the runner closed
	g.Restart(startContext(t), src) // spawn sees the runner closed

	assert.Never(t, func() bool { return h.opens() > 0 }, 200*time.Millisecond, eventuallyInterval,
		"no harvester should start after shutdown")
	require.False(t, g.hasID(g.identifier.ID(src)))
}

// TestHarvesterRunner_StopWhileQueuedOnLimit asserts a source cancelled while it
// is waiting for a harvester-limit permit is torn down without ever reading.
func TestHarvesterRunner_StopWhileQueuedOnLimit(t *testing.T) {
	// src1 holds the only slot; src2 is queued (statusWaiting, no goroutine), so
	// readFn only ever runs for src1 — src2 never opens a session while queued.
	h := &fakeHarvester{readFn: blockUntilCancelled}
	g := testHarvesterRunner(t, h, 1) // single open-file slot

	g.start()
	src1 := &testSource{name: "/path/to/test/1"}
	src2 := &testSource{name: "/path/to/test/2"}
	id2 := g.identifier.ID(src2)
	g.Start(startContext(t), src1)
	requireEventually(t, func() bool { return h.opens() == 1 }, "src1 should hold the only slot")

	g.Start(startContext(t), src2) // no slot: queued
	requireEventually(t, func() bool {
		s, ok := g.statusOf(id2)
		return ok && s == statusWaiting
	}, "src2 should be queued in statusWaiting")

	g.Stop(src2) // stop while queued: must tear down without opening a session
	requireEventually(t, func() bool { return !g.hasID(id2) },
		"src2 should be removed while still queued")
	require.Equal(t, 1, h.opens(), "src2 must not open a session while queued")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_RunSkipsAlreadyActive asserts run does not spawn a second
// reader for a source that is already running or being torn down.
func TestHarvesterRunner_RunSkipsAlreadyActive(t *testing.T) {
	g := testHarvesterRunner(t, &fakeHarvester{}, 0)

	// A reader goroutine already owns a running source: run must be a no-op (no
	// new goroutine, status unchanged) and leave teardown to that reader.
	state := &sourceState{srcID: "x", ctx: startContext(t), status: statusRunning, done: make(chan struct{})}
	g.mu.Lock()
	g.states["x"] = state
	g.mu.Unlock()

	g.run(state)

	got, _ := g.statusOf("x")
	require.Equal(t, statusRunning, got, "run must not disturb a source a reader already owns")

	g.mu.Lock()
	delete(g.states, "x")
	g.mu.Unlock()

	// A finished source is skipped: finish() is idempotent, so run must not panic
	// or re-tear-down.
	fin := &sourceState{srcID: "y", ctx: startContext(t), finished: true, done: make(chan struct{})}
	g.run(fin)
	require.False(t, g.hasID("y"))

	// A closing source with no holder must be torn down by run().
	closing := &sourceState{srcID: "z", ctx: startContext(t), status: statusClosing, done: make(chan struct{})}
	g.mu.Lock()
	g.states["z"] = closing
	g.mu.Unlock()
	g.run(closing)
	require.False(t, g.hasID("z"), "run must finish a closing source with no holder")
}

// TestHarvesterRunner_StopFinishesNewSource asserts that stopping a source that
// is still in statusNew (registered by enqueue but not yet picked up by run)
// tears it down instead of leaking it.
func TestHarvesterRunner_StopFinishesNewSource(t *testing.T) {
	g := testHarvesterRunner(t, &fakeHarvester{}, 0)
	src := &testSource{name: "/path/to/new"}
	id := g.identifier.ID(src)

	_, cancel := context.WithCancel(context.Background())
	state := &sourceState{
		srcID:  id,
		src:    src,
		ctx:    startContext(t),
		cancel: cancel,
		status: statusNew,
		done:   make(chan struct{}),
	}
	g.mu.Lock()
	g.states[id] = state
	g.nActive++ // enqueue counts statusNew as active
	g.mu.Unlock()

	g.Stop(src)

	require.False(t, g.hasID(id), "a stopped new source must be removed")
	select {
	case <-state.done:
	default:
		t.Fatal("done channel must be closed after stopping a new source")
	}
	active, parked := g.counts()
	require.Equal(t, 0, active, "active gauge must return to zero")
	require.Equal(t, 0, parked, "parked gauge must return to zero")
}

// TestHarvesterRunner_StopAndWaitFinishesNewSource asserts that stopAndWait (the
// path Restart uses) does not block forever on a statusNew source whose done
// channel will never be closed by a reader.
func TestHarvesterRunner_StopAndWaitFinishesNewSource(t *testing.T) {
	g := testHarvesterRunner(t, &fakeHarvester{}, 0)
	id := "new-src"

	_, cancel := context.WithCancel(context.Background())
	state := &sourceState{
		srcID:  id,
		ctx:    startContext(t),
		cancel: cancel,
		status: statusNew,
		done:   make(chan struct{}),
	}
	g.mu.Lock()
	g.states[id] = state
	g.nActive++
	g.mu.Unlock()

	done := make(chan struct{})
	go func() {
		g.stopAndWait(state)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(eventuallyTimeout):
		t.Fatal("stopAndWait must not block on a statusNew source")
	}
	require.False(t, g.hasID(id), "stopAndWait must remove the new source")
}

// TestHarvesterRunner_RunFinishesSourceClosedDuringPollHandoff asserts run()
// tears down a polling source when Stop races the PollResume hand-off (leak fix).
func TestHarvesterRunner_RunFinishesSourceClosedDuringPollHandoff(t *testing.T) {
	h := parkYieldResumeHarvester()
	g := testHarvesterRunner(t, h, 0)
	src := &testSource{name: "/path/to/test"}
	id, state := startParkedAndClaimDue(t, g, src)

	g.Stop(src)
	require.True(t, g.hasID(id), "Stop must hand a polling source to its holder, not finish it")
	s, _ := g.statusOf(id)
	require.Equal(t, statusClosing, s)

	g.run(state)
	requireEventually(t, func() bool { return !g.hasID(id) },
		"run() must finish a source closed during the poll hand-off")
	require.True(t, h.lastSession().isClosed(), "session must be closed on teardown")
	active, parked := g.counts()
	require.Equal(t, 0, active, "active gauge must return to zero")
	require.Equal(t, 0, parked, "parked gauge must return to zero")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_StopAndWaitUnblockedByPollHandoff asserts run() unblocks
// stopAndWait when Stop races the PollResume hand-off (deadlock fix).
func TestHarvesterRunner_StopAndWaitUnblockedByPollHandoff(t *testing.T) {
	g := testHarvesterRunner(t, parkYieldResumeHarvester(), 0)
	src := &testSource{name: "/path/to/test"}
	id, state := startParkedAndClaimDue(t, g, src)

	done := make(chan struct{})
	go func() {
		g.stopAndWait(state) // sets statusClosing, then blocks on <-state.done
		close(done)
	}()

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusClosing
	}, "stopAndWait should mark the source closing and wait for its holder")

	g.run(state)

	select {
	case <-done:
	case <-time.After(eventuallyTimeout):
		t.Fatal("stopAndWait deadlocked: run() did not finish the polling source")
	}
	require.False(t, g.hasID(id))

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_ReadUntilEOFDrainsParkedOnStop asserts that, with
// read_until_eof enabled, StopHarvesters reads a parked source again (drains it
// to EOF) before tearing it down.
func TestHarvesterRunner_ReadUntilEOFDrainsParkedOnStop(t *testing.T) {
	var reads atomic.Int64
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) {
			reads.Add(1)
			return SliceYield, nil // always caught up to EOF
		},
		pollFn: func(int) PollResult { return PollPark },
	}
	g := testHarvesterRunnerEOF(t, h, 0, ReadUntilEOFConfig{Enabled: true, Timeout: 5 * time.Second})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked
	}, "source should park after its first read")
	readsBeforePark := reads.Load()

	require.NoError(t, g.StopHarvesters())
	require.Greater(t, reads.Load(), readsBeforePark,
		"a parked source should be drained (read again) on read_until_eof stop")
	require.False(t, g.hasID(id))
	require.True(t, h.lastSession().isClosed())
}

// TestHarvesterRunner_NoDrainWhenDisabled asserts that with read_until_eof
// disabled a parked source is torn down on stop without an extra read.
func TestHarvesterRunner_NoDrainWhenDisabled(t *testing.T) {
	var reads atomic.Int64
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) {
			reads.Add(1)
			return SliceYield, nil
		},
		pollFn: func(int) PollResult { return PollPark },
	}
	g := testHarvesterRunner(t, h, 0) // read_until_eof disabled

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked
	}, "source should park")
	readsBeforeStop := reads.Load()

	require.NoError(t, g.StopHarvesters())
	require.Equal(t, readsBeforeStop, reads.Load(), "no drain read when read_until_eof is disabled")
	require.False(t, g.hasID(id))
}

// TestHarvesterRunner_ReadUntilEOFTimeoutBoundsDrain asserts that a source stuck
// mid-read is cancelled after the configured Timeout so shutdown cannot hang.
func TestHarvesterRunner_ReadUntilEOFTimeoutBoundsDrain(t *testing.T) {
	h := &fakeHarvester{readFn: blockUntilCancelled} // never reaches EOF on its own
	g := testHarvesterRunnerEOF(t, h, 0, ReadUntilEOFConfig{Enabled: true, Timeout: 200 * time.Millisecond})

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src := &testSource{name: "/path/to/test"}
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool { return h.opens() == 1 }, "source should be reading")

	start := time.Now()
	require.NoError(t, g.StopHarvesters())
	require.WithinDuration(t, start, time.Now(), 5*time.Second,
		"a stuck drain must be bounded by the timeout, not hang")
	require.False(t, g.hasID(id))
}

// TestGrowBackoff covers the backoff growth, including the cap at the maximum.
func TestGrowBackoff(t *testing.T) {
	require.Equal(t, minWakerBackoff, growBackoff(0), "non-positive grows to the minimum")
	require.Equal(t, minWakerBackoff, growBackoff(-time.Second), "negative grows to the minimum")
	require.Equal(t, 2*minWakerBackoff, growBackoff(minWakerBackoff), "doubles below the cap")
	require.Equal(t, maxWakerBackoff, growBackoff(maxWakerBackoff), "is capped at the maximum")
	require.Equal(t, maxWakerBackoff, growBackoff(maxWakerBackoff-time.Millisecond),
		"doubling past the cap clamps to the maximum")
}

// --- Test scaffolding ---------------------------------------------------

// testHarvesterRunner builds a harvesterRunner wired to controllable fakes and a
// fresh in-memory store and metrics.
func testHarvesterRunner(t *testing.T, h Harvester, limit uint64) *harvesterRunner {
	t.Helper()
	return testHarvesterRunnerEOF(t, h, limit, ReadUntilEOFConfig{})
}

func testHarvesterRunnerEOF(t *testing.T, h Harvester, limit uint64, eof ReadUntilEOFConfig) *harvesterRunner {
	t.Helper()
	logger := logptest.NewTestingLogger(t, "")
	ident, err := NewSourceIdentifier("filestream", "")
	require.NoError(t, err)

	runnerCtx := v2.Context{Logger: logger, Cancelation: context.Background()}
	return newHarvesterRunner(
		runnerCtx,
		limit,
		&MockPipeline{},
		h,
		5*time.Second,
		testOpenStore(t, "test", nil),
		nil, // ackCH: the fake sessions never publish, so no ACKs are produced
		ident,
		NewMetrics(monitoring.NewRegistry(), logger),
		"test-input",
		eof,
	)
}

// startContext returns the input.Context passed to Start/Restart. The runner
// replaces its Cancelation with a per-source cancel context, so Background is
// fine here.
func startContext(t *testing.T) v2.Context {
	t.Helper()
	return v2.Context{Logger: logptest.NewTestingLogger(t, ""), Cancelation: context.Background()}
}

// parkYieldResumeHarvester parks on yield and reports new data on poll.
func parkYieldResumeHarvester() *fakeHarvester {
	return &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) { return SliceYield, nil },
		pollFn: func(_ int) PollResult { return PollResume },
	}
}

// startParkedAndClaimDue starts src, waits until parked, then pops it as
// statusPolling (waker hand-off point before pollParked/run).
func startParkedAndClaimDue(t *testing.T, g *harvesterRunner, src *testSource) (id string, state *sourceState) {
	t.Helper()
	id = g.identifier.ID(src)
	g.Start(startContext(t), src)
	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked
	}, "source should park after a yielding read")
	state = g.popDueNow()
	require.NotNil(t, state, "the parked source should be due")
	return id, state
}

// blockUntilCancelled is a readFn that blocks until the source's context is
// cancelled (by Stop/Restart/StopHarvesters), modelling a long-running read.
func blockUntilCancelled(_ int, ctx v2.Context) (SliceVerdict, error) {
	<-ctx.Cancelation.Done()
	return SliceDone, nil
}

// fakeHarvester is a controllable Harvester. Each OpenSession produces a
// fakeSession driven by readFn/pollFn; openErr forces OpenSession to fail.
type fakeHarvester struct {
	mu       sync.Mutex
	openErr  error
	gzip     bool // sessions report IsGZIP() == true
	readFn   func(call int, ctx v2.Context) (SliceVerdict, error)
	pollFn   func(call int) PollResult
	sessions []*fakeSession
}

func (h *fakeHarvester) Name() string                          { return "fake" }
func (h *fakeHarvester) Test(_ Source, _ v2.TestContext) error { return nil }

func (h *fakeHarvester) OpenSession(
	_ v2.Context, _ Source, _ Cursor, _ *Metrics,
) (HarvesterSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.openErr != nil {
		return nil, h.openErr
	}
	s := &fakeSession{readFn: h.readFn, pollFn: h.pollFn, gzip: h.gzip}
	h.sessions = append(h.sessions, s)
	return s, nil
}

func (h *fakeHarvester) opens() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.sessions)
}

func (h *fakeHarvester) session(i int) *fakeSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[i]
}

func (h *fakeHarvester) lastSession() *fakeSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[len(h.sessions)-1]
}

// fakeSession is a controllable HarvesterSession. readFn/pollFn receive a 1-based
// call counter so a test can vary behaviour across calls. With no readFn a read
// returns SliceDone; with no pollFn a poll returns PollClose.
type fakeSession struct {
	mu     sync.Mutex
	reads  int
	polls  int
	offset int64
	closed bool
	gzip   bool

	readFn func(call int, ctx v2.Context) (SliceVerdict, error)
	pollFn func(call int) PollResult
}

func (s *fakeSession) ReadSlice(ctx v2.Context, _ Publisher) (SliceVerdict, error) {
	s.mu.Lock()
	s.reads++
	call := s.reads
	fn := s.readFn
	s.mu.Unlock()

	verdict, err := SliceDone, error(nil)
	if fn != nil {
		verdict, err = fn(call, ctx)
	}

	// A yielding read models having consumed available data, so advance the
	// offset: the runner uses progress to pick the park backoff.
	if err == nil && verdict == SliceYield {
		s.mu.Lock()
		s.offset++
		s.mu.Unlock()
	}
	return verdict, err
}

func (s *fakeSession) Poll() PollResult {
	s.mu.Lock()
	s.polls++
	call := s.polls
	fn := s.pollFn
	s.mu.Unlock()
	if fn != nil {
		return fn(call)
	}
	return PollClose
}

func (s *fakeSession) Offset() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offset
}

func (s *fakeSession) IsGZIP() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gzip
}

func (s *fakeSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *fakeSession) readCount() int { s.mu.Lock(); defer s.mu.Unlock(); return s.reads }
func (s *fakeSession) pollCount() int { s.mu.Lock(); defer s.mu.Unlock(); return s.polls }
func (s *fakeSession) isClosed() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.closed }

// --- Test-only inspection helpers on harvesterRunner --------------------

func (g *harvesterRunner) hasID(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.states[id]
	return ok
}

func (g *harvesterRunner) stateFor(id string) (*sourceState, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, ok := g.states[id]
	return state, ok
}

func (g *harvesterRunner) statusOf(id string) (sourceStatus, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, ok := g.states[id]
	if !ok {
		return 0, false
	}
	return state.status, true
}

func (g *harvesterRunner) parkedLen() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.parked.Len()
}

func (g *harvesterRunner) counts() (active, parked int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nActive, g.nParked
}

func (g *harvesterRunner) countWaiting() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nWaiting
}

func (g *harvesterRunner) countStates() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.states)
}

// popDueNow pops the single parked source as if its backoff had elapsed,
// returning it claimed (statusPolling) and ready for pollParked.
func (g *harvesterRunner) popDueNow() *sourceState {
	g.mu.Lock()
	defer g.mu.Unlock()
	due := g.popDue(time.Now().Add(2 * maxWakerBackoff))
	if len(due) == 0 {
		return nil
	}
	return due[0]
}

// --- Reusable pipeline/client fakes -------------------------------------

// MockClient is a minimal beat.Client that records published events and
// acknowledges any update operations on them.
type MockClient struct {
	mu        sync.Mutex
	published []beat.Event
	closed    bool
}

func (m *MockClient) Publish(e beat.Event) { m.PublishAll([]beat.Event{e}) }

func (m *MockClient) PublishAll(es []beat.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, evt := range es {
		if op, ok := evt.Private.(*updateOp); ok {
			op.done(1)
		}
	}
	m.published = append(m.published, es...)
}

func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return fmt.Errorf("mock already closed")
	}
	m.closed = true
	return nil
}

// MockPipeline is a minimal beat.PipelineConnector. connectErr, when set, makes
// ConnectWith fail (used to exercise the runner's setup-error path).
type MockPipeline struct {
	mu         sync.Mutex
	c          beat.Client
	connectErr error
}

func (mp *MockPipeline) ConnectWith(_ beat.ClientConfig) (beat.Client, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if mp.connectErr != nil {
		return nil, mp.connectErr
	}
	c := &MockClient{}
	mp.c = c
	return c, nil
}

func (mp *MockPipeline) Connect() (beat.Client, error) { return mp.ConnectWith(beat.ClientConfig{}) }

func (mp *MockPipeline) Disconnect(_ context.Context) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if mp.c == nil {
		return nil
	}
	return mp.c.Close()
}

// recordingStatusReporter is a status.StatusReporter that captures the last
// reported status and message, so a test can assert the input was degraded.
type recordingStatusReporter struct {
	mu     sync.Mutex
	status status.Status
	msg    string
	set    bool
}

func (r *recordingStatusReporter) UpdateStatus(s status.Status, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = s
	r.msg = msg
	r.set = true
}

func (r *recordingStatusReporter) last() status.Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.set {
		return status.Unknown
	}
	return r.status
}

func (r *recordingStatusReporter) lastMsg() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.msg
}
