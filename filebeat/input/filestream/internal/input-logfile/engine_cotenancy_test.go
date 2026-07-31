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

// TestSharedEngine_PollsDueSourcesConcurrently asserts that sources coming due
// together are polled together, across inputs, with the grace period on the
// batch rather than on each source in turn.
//
// Poll is typically a stat(), and on an unresponsive mount it hangs for the
// whole grace period. Polled in turn, one such source would delay every other
// input's sources on the goroutine they all share.
func TestSharedEngine_PollsDueSourcesConcurrently(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	engine := newEngine(logger)
	t.Cleanup(engine.stop)

	// Well under the grace period, so a concurrent batch finishes within it and
	// the test never depends on the abandonment path.
	const holdFor = pollGracePeriod / 4
	const sourcesPerInput = 4

	// inFlightA/inFlightB count the polls each input has in progress; maxInFlight
	// is the largest overlap seen across both, and crossInput records whether the
	// two inputs were ever polled at the same instant.
	var inFlightA, inFlightB, maxInFlight atomic.Int64
	var crossInput atomic.Bool
	hold := func(mine, theirs *atomic.Int64, polls *atomic.Int64) *fakeHarvester {
		h := pollCountingHarvester(polls, 0)
		inner := h.pollFn
		h.pollFn = func(call int) PollResult {
			n := mine.Add(1) + theirs.Load()
			for {
				observed := maxInFlight.Load()
				if n <= observed || maxInFlight.CompareAndSwap(observed, n) {
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
	// Long enough that the sources of both inputs park before the first of them
	// comes due, so they land in one due batch.
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

	// Polled in turn, maxInFlight never leaves 1 and no two inputs are ever in
	// Poll at once, so neither condition is a timing threshold to be tuned. A
	// straggler that misses one batch only delays the observation to the next
	// park cycle.
	requireEventually(t,
		func() bool { return crossInput.Load() && maxInFlight.Load() >= int64(sourcesPerInput) },
		"sources that come due together must be polled together, across inputs, not one after another")

	require.NoError(t, runnerA.StopHarvesters())
	require.NoError(t, runnerB.StopHarvesters())
}
