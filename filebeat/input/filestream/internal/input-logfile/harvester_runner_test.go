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

// TestHarvesterRunner_RespectsHarvesterLimit asserts that with harvester_limit=1
// only one source reads at a time: a second source's reader blocks on the
// semaphore before it even opens its session, and only proceeds once the first
// finishes.
func TestHarvesterRunner_RespectsHarvesterLimit(t *testing.T) {
	release := make(chan struct{})
	h := &fakeHarvester{
		readFn: func(_ int, _ v2.Context) (SliceVerdict, error) {
			<-release // hold the single permit until the test releases it
			return SliceDone, nil
		},
	}
	g := testHarvesterRunner(t, h, 1)

	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.WaitUntilOriginalCount()

	g.start()
	src1 := &testSource{name: "/path/to/test/1"}
	src2 := &testSource{name: "/path/to/test/2"}
	g.Start(startContext(t), src1)
	g.Start(startContext(t), src2)

	// One source acquires the permit and opens its session; the other must wait.
	requireEventually(t, func() bool { return h.opens() == 1 },
		"exactly one harvester should run under the limit")
	assert.Never(t, func() bool { return h.opens() > 1 }, 200*time.Millisecond, eventuallyInterval,
		"the second harvester must not open a session while the limit is reached")

	// Release the first; the second now acquires the permit and runs.
	close(release)
	requireEventually(t, func() bool {
		return h.opens() == 2 &&
			!g.hasID(g.identifier.ID(src1)) && !g.hasID(g.identifier.ID(src2))
	}, "the second harvester should run and both should tear down")

	require.NoError(t, g.StopHarvesters())
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
	ps := g.popDueNow()
	require.NotNil(t, ps, "the parked source should be due")
	g.pollParked(ps)

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
	id := g.identifier.ID(src)
	g.Start(startContext(t), src)

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked
	}, "source should park")

	ps := g.popDueNow()
	require.NotNil(t, ps)
	g.pollParked(ps)

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

	ps, ok := g.stateFor(id)
	require.True(t, ok)
	require.Equal(t, minWakerBackoff, ps.backoff, "a progressing read parks with the minimum backoff")

	ps = g.popDueNow()
	require.NotNil(t, ps)
	g.pollParked(ps)

	requireEventually(t, func() bool {
		s, ok := g.statusOf(id)
		return ok && s == statusParked && ps.backoff == growBackoff(minWakerBackoff)
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
	release := make(chan struct{})
	defer close(release)
	h := &fakeHarvester{
		readFn: func(_ int, ctx v2.Context) (SliceVerdict, error) {
			select {
			case <-release:
			case <-ctx.Cancelation.Done():
			}
			return SliceDone, nil
		},
	}
	g := testHarvesterRunner(t, h, 1) // single permit

	g.start()
	src1 := &testSource{name: "/path/to/test/1"}
	src2 := &testSource{name: "/path/to/test/2"}
	g.Start(startContext(t), src1)
	requireEventually(t, func() bool { return h.opens() == 1 }, "src1 should hold the only permit")

	g.Start(startContext(t), src2) // queues on the semaphore
	requireEventually(t, func() bool { return g.hasID(g.identifier.ID(src2)) }, "src2 should be registered")

	g.Stop(src2) // cancel while queued: must tear down without opening a session
	requireEventually(t, func() bool { return !g.hasID(g.identifier.ID(src2)) },
		"src2 should be removed while still queued")
	require.Equal(t, 1, h.opens(), "src2 must not open a session while queued")

	require.NoError(t, g.StopHarvesters())
}

// TestHarvesterRunner_RunSkipsAlreadyActive asserts run does not spawn a second
// reader for a source that is already running or being torn down.
func TestHarvesterRunner_RunSkipsAlreadyActive(t *testing.T) {
	g := testHarvesterRunner(t, &fakeHarvester{}, 0)

	for _, st := range []sourceStatus{statusRunning, statusClosing} {
		ps := &sourceState{srcID: "x", status: st, done: make(chan struct{})}
		g.mu.Lock()
		g.states["x"] = ps
		g.mu.Unlock()

		g.run(ps) // must be a no-op: no new goroutine, status unchanged

		got, _ := g.statusOf("x")
		require.Equal(t, st, got, "run must not change the status of an already-active source")

		g.mu.Lock()
		delete(g.states, "x")
		g.mu.Unlock()
	}
	// A finished source is also skipped.
	ps := &sourceState{srcID: "y", finished: true, done: make(chan struct{})}
	g.run(ps)
	require.False(t, g.hasID("y"))
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
	s := &fakeSession{readFn: h.readFn, pollFn: h.pollFn}
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
	ps, ok := g.states[id]
	return ps, ok
}

func (g *harvesterRunner) statusOf(id string) (sourceStatus, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	ps, ok := g.states[id]
	if !ok {
		return 0, false
	}
	return ps.status, true
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
