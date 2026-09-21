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

package input_logfile

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// pollCountingHarvester returns a harvester whose sources park on their first
// read and stay parked, counting every poll the scheduler makes and taking
// pollDelay to answer each one.
func pollCountingHarvester(polls *atomic.Int64, pollDelay time.Duration) *fakeHarvester {
	return &fakeHarvester{
		readFn: func(int, v2.Context) (SliceVerdict, error) { return SliceYield, nil },
		pollFn: func(int) PollResult {
			polls.Add(1)
			if pollDelay > 0 {
				time.Sleep(pollDelay)
			}
			return PollPark
		},
	}
}

// fastBackoff re-parks a source almost immediately, so a test observes many
// scheduler passes in a short window instead of waiting out the default backoff.
var fastBackoff = BackoffConfig{Init: time.Millisecond, Max: 2 * time.Millisecond}

// TestSharedEngine_PollsEachInputsSources asserts that with two inputs on one
// scheduler, a source parked by either is polled through its own runner — the
// routing that keeps its events going to its own input's pipeline client.
func TestSharedEngine_PollsEachInputsSources(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	engine := newEngine(logger)
	t.Cleanup(engine.stop)

	var pollsA, pollsB atomic.Int64
	runnerA := newHarvesterRunnerOn(t, engine, func() {}, pollCountingHarvester(&pollsA, 0), 0, ReadUntilEOFConfig{})
	runnerB := newHarvesterRunnerOn(t, engine, func() {}, pollCountingHarvester(&pollsB, 0), 0, ReadUntilEOFConfig{})
	runnerA.backoff, runnerB.backoff = fastBackoff, fastBackoff

	runnerA.start()
	runnerB.start()
	require.Same(t, runnerA.engine, runnerB.engine, "both inputs must be on one scheduler")

	runnerA.Start(startContext(t), &testSource{name: "/a"})
	runnerB.Start(startContext(t), &testSource{name: "/b"})

	requireEventually(t, func() bool { return pollsA.Load() > 0 && pollsB.Load() > 0 },
		"one scheduler must poll the sources of both inputs")

	require.NoError(t, runnerA.StopHarvesters())
	require.NoError(t, runnerB.StopHarvesters())
}

// TestSharedEngine_StoppingOneInputLeavesTheOtherRunning asserts StopHarvesters
// tears down only its own input's sources: an input shutting down must not stop
// the inputs it shares a scheduler with from reading.
func TestSharedEngine_StoppingOneInputLeavesTheOtherRunning(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	engine := newEngine(logger)
	t.Cleanup(engine.stop)

	var pollsA, pollsB atomic.Int64
	runnerA := newHarvesterRunnerOn(t, engine, func() {}, pollCountingHarvester(&pollsA, 0), 0, ReadUntilEOFConfig{})
	runnerB := newHarvesterRunnerOn(t, engine, func() {}, pollCountingHarvester(&pollsB, 0), 0, ReadUntilEOFConfig{})
	runnerA.backoff, runnerB.backoff = fastBackoff, fastBackoff

	runnerA.start()
	runnerB.start()
	runnerA.Start(startContext(t), &testSource{name: "/a"})
	runnerB.Start(startContext(t), &testSource{name: "/b"})
	requireEventually(t, func() bool { return pollsA.Load() > 0 && pollsB.Load() > 0 },
		"both inputs must be running before one stops")

	require.NoError(t, runnerA.StopHarvesters())
	stoppedAt := pollsA.Load()
	resumeFrom := pollsB.Load()

	requireEventually(t, func() bool { return pollsB.Load() > resumeFrom+5 },
		"the surviving input must keep being polled after its co-tenant stops")

	assert.LessOrEqual(t, pollsA.Load(), stoppedAt+1,
		"a stopped input's sources must not keep being polled")

	require.NoError(t, runnerB.StopHarvesters())
}

// TestSharedEngine_PollsSerialPerInputConcurrentAcrossInputs asserts the shape of
// the scheduling: an input's due sources are polled one at a time, and different
// inputs poll at the same time. So the scheduler's fan-out is one chain per input
// with due work, not a goroutine per due file, and no input waits on another.
//
// This is what a per-input waker gave before the engine was shared; the engine
// keeps it while costing one sleeping goroutine for the process instead of N.
func TestSharedEngine_PollsSerialPerInputConcurrentAcrossInputs(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	engine := newEngine(logger)
	t.Cleanup(engine.stop)

	// Well under the grace period, so no poll is ever abandoned and every overlap
	// the test sees is real concurrency rather than a stuck poll left behind.
	const holdFor = pollGracePeriod / 4
	const sourcesPerInput = 4

	// inFlightA/inFlightB count the polls each input has in progress; maxPerInput
	// is the largest either reached on its own, and crossInput records whether the
	// two inputs were ever in Poll at the same instant.
	var inFlightA, inFlightB, maxPerInput atomic.Int64
	var crossInput atomic.Bool
	hold := func(mine, theirs, polls *atomic.Int64) *fakeHarvester {
		h := pollCountingHarvester(polls, 0)
		inner := h.pollFn
		h.pollFn = func(call int) PollResult {
			n := mine.Add(1)
			for {
				observed := maxPerInput.Load()
				if n <= observed || maxPerInput.CompareAndSwap(observed, n) {
					break
				}
			}
			if theirs.Load() > 0 {
				crossInput.Store(true)
			}
			time.Sleep(holdFor)
			mine.Add(-1)
			return inner(call)
		}
		return h
	}

	var pollsA, pollsB atomic.Int64
	// Long enough that every source parks before the first comes due, so each
	// input has a queue of sources to work through rather than one at a time.
	batching := BackoffConfig{Init: 250 * time.Millisecond, Max: 250 * time.Millisecond}
	runnerA := newHarvesterRunnerOn(t, engine, func() {}, hold(&inFlightA, &inFlightB, &pollsA), 0, ReadUntilEOFConfig{})
	runnerB := newHarvesterRunnerOn(t, engine, func() {}, hold(&inFlightB, &inFlightA, &pollsB), 0, ReadUntilEOFConfig{})
	runnerA.backoff, runnerB.backoff = batching, batching

	runnerA.start()
	runnerB.start()
	for i := range sourcesPerInput {
		runnerA.Start(startContext(t), &testSource{name: fmt.Sprintf("/a-%d", i)})
		runnerB.Start(startContext(t), &testSource{name: fmt.Sprintf("/b-%d", i)})
	}

	// Inputs overlapping each other is the property; a straggler that misses one
	// batch only delays the observation to the next park cycle.
	requireEventually(t, crossInput.Load,
		"different inputs must poll at the same time, not one input's queue after another's")
	// Serial within an input is an invariant, so it is checked for the whole run
	// rather than at an instant: two of one input's sources in Poll at once means
	// a second chain was started for it.
	assert.Never(t, func() bool { return maxPerInput.Load() > 1 }, 500*time.Millisecond, eventuallyInterval,
		"an input must poll its due sources in turn, never two at once")

	require.NoError(t, runnerA.StopHarvesters())
	require.NoError(t, runnerB.StopHarvesters())
}

// TestSharedEngine_StuckPollIsAbandonedAndOthersContinue asserts what happens
// when a Poll does not come back — a stat() on an unresponsive mount. It is left
// running on its own goroutine after the grace period, and neither the rest of
// its own input's queue nor the other input on the same scheduler waits for it.
func TestSharedEngine_StuckPollIsAbandonedAndOthersContinue(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	engine := newEngine(logger)
	t.Cleanup(engine.stop)

	const sourcesPerInput = 3

	// The first poll of input A never returns; every later one is healthy, so any
	// progress A makes proves the chain moved past the stuck source.
	release := make(chan struct{})
	var wedged, stuckLive atomic.Bool
	var pollsA, pollsB atomic.Int64
	hA := pollCountingHarvester(&pollsA, 0)
	inner := hA.pollFn
	hA.pollFn = func(call int) PollResult {
		if wedged.CompareAndSwap(false, true) {
			stuckLive.Store(true)
			<-release
			stuckLive.Store(false)
			return PollPark
		}
		return inner(call)
	}

	runnerA := newHarvesterRunnerOn(t, engine, func() {}, hA, 0, ReadUntilEOFConfig{})
	runnerB := newHarvesterRunnerOn(t, engine, func() {}, pollCountingHarvester(&pollsB, 0), 0, ReadUntilEOFConfig{})
	runnerA.backoff, runnerB.backoff = fastBackoff, fastBackoff

	runnerA.start()
	runnerB.start()
	for i := range sourcesPerInput {
		runnerA.Start(startContext(t), &testSource{name: fmt.Sprintf("/a-%d", i)})
		runnerB.Start(startContext(t), &testSource{name: fmt.Sprintf("/b-%d", i)})
	}

	// Both inputs must keep being polled while the stuck poll is still sitting in
	// its goroutine, so neither is waiting on it.
	requireEventually(t, func() bool { return pollsA.Load() > 0 && pollsB.Load() > 5 },
		"a stuck poll must not hold up its own input's other sources or another input")
	assert.True(t, stuckLive.Load(), "the stuck poll must still be running, i.e. it was abandoned rather than waited out")

	close(release)
	require.NoError(t, runnerA.StopHarvesters())
	require.NoError(t, runnerB.StopHarvesters())
}
