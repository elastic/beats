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

package hbscheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func testParams() Params {
	return Params{
		Limit:          10,
		Registry:       monitoring.NewRegistry(),
		Location:       time.UTC,
		JobLimitByType: map[string]*config.JobLimit{"browser": {Limit: 2}},
	}
}

// runOnceSchedule runs a job immediately and then effectively never again, so
// that tests observe exactly one run per job.
type runOnceSchedule struct{}

func (runOnceSchedule) RunOnInit() bool              { return true }
func (runOnceSchedule) Next(now time.Time) time.Time { return now.Add(time.Hour) }

// blockingJob adds a job that reports the id it ran with and then blocks until
// unblock is closed, so that tests can observe which jobs hold a slot.
func blockingJob(t *testing.T, sched *scheduler.Scheduler, id string, started chan<- string, unblock <-chan struct{}) {
	t.Helper()

	_, err := sched.Add(runOnceSchedule{}, nil, id, func(_ context.Context) []scheduler.TaskFunc {
		started <- id
		<-unblock
		return nil
	}, "browser")
	require.NoError(t, err)
}

func mustAcquire(t *testing.T, g *groups, logger *logp.Logger, group string, params Params) (*scheduler.Scheduler, ReleaseFunc) {
	t.Helper()
	sched, release, err := g.acquire(logger, group, params)
	require.NoError(t, err)
	return sched, release
}

// neverSchedule never runs its job, it is only used to probe the scheduler.
type neverSchedule struct{}

func (neverSchedule) RunOnInit() bool              { return false }
func (neverSchedule) Next(now time.Time) time.Time { return now.Add(24 * time.Hour) }

// stopped reports whether sched was stopped, going by whether it still accepts
// jobs.
func stopped(t *testing.T, sched *scheduler.Scheduler) bool {
	t.Helper()

	remove, err := sched.Add(neverSchedule{}, nil, "probe", func(context.Context) []scheduler.TaskFunc { return nil }, "")
	if errors.Is(err, scheduler.ErrAlreadyStopped) {
		return true
	}
	require.NoError(t, err)
	remove()
	return false
}

func TestAcquireSharesOneSchedulerPerGroup(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	first, releaseFirst := mustAcquire(t, g, logger, "group", testParams())
	second, releaseSecond := mustAcquire(t, g, logger, "group", testParams())

	assert.Same(t, first, second, "consumers of a group must get the same scheduler")
	assert.False(t, stopped(t, first))

	releaseFirst()
	assert.False(t, stopped(t, first), "the scheduler must keep running while its group has consumers")

	releaseSecond()
	assert.True(t, stopped(t, first), "the scheduler must stop once its last consumer released it")
}

func TestGroupsAreIsolated(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	// The empty group is the default one, used by a standalone Heartbeat.
	defaultSched, releaseDefault := mustAcquire(t, g, logger, "", testParams())
	first, releaseFirst := mustAcquire(t, g, logger, "first", testParams())
	second, releaseSecond := mustAcquire(t, g, logger, "second", testParams())
	defer releaseSecond()

	assert.NotSame(t, defaultSched, first)
	assert.NotSame(t, defaultSched, second)
	assert.NotSame(t, first, second)

	releaseDefault()
	releaseFirst()
	assert.True(t, stopped(t, defaultSched))
	assert.True(t, stopped(t, first))
	assert.False(t, stopped(t, second), "releasing one group must not stop another group's scheduler")
}

func TestReleaseIsIdempotent(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	sched, releaseFirst := mustAcquire(t, g, logger, "group", testParams())
	_, releaseSecond := mustAcquire(t, g, logger, "group", testParams())

	releaseFirst()
	releaseFirst()
	releaseFirst()
	assert.False(t, stopped(t, sched), "repeated releases must not drop another consumer's reference")

	releaseSecond()
	assert.True(t, stopped(t, sched))
}

func TestAcquireAfterLastReleaseCreatesNewScheduler(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	first, releaseFirst := mustAcquire(t, g, logger, "group", testParams())
	releaseFirst()
	require.True(t, stopped(t, first))

	second, releaseSecond := mustAcquire(t, g, logger, "group", testParams())
	defer releaseSecond()

	assert.NotSame(t, first, second, "a stopped scheduler cannot be restarted, so a new one is needed")
	assert.False(t, stopped(t, second))
}

func TestAcquireWarnsAboutConflictingParams(t *testing.T) {
	g := newGroups()
	logger, observed := logptest.NewTestingLoggerWithObserver(t, "")

	_, release := mustAcquire(t, g, logger, "group", testParams())
	defer release()

	conflicting := testParams()
	conflicting.Limit = 20
	conflicting.Location = time.FixedZone("Fake", 3600)
	conflicting.JobLimitByType = map[string]*config.JobLimit{"http": {Limit: 4}}

	_, releaseConflicting := mustAcquire(t, g, logger, "group", conflicting)
	defer releaseConflicting()

	warnings := observed.FilterLevelExact(logp.WarnLevel.ZapLevel()).All()
	require.Len(t, warnings, 1, "a conflicting consumer must be warned about once")
	assert.Contains(t, warnings[0].Message,
		"running with {Limit:10 Location:UTC JobLimits:map[browser:2]}, "+
			"requested {Limit:20 Location:Fake JobLimits:map[http:4]}",
		"the warning must describe both the running and the requested settings")
}

func TestAcquireErrorsOnRunOnceConflict(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	_, release := mustAcquire(t, g, logger, "group", testParams()) // run_once=false
	defer release()

	runOnce := testParams()
	runOnce.RunOnce = true
	_, _, err := g.acquire(logger, "group", runOnce)
	require.Error(t, err, "a run_once consumer must not join a group without run_once")
	assert.Contains(t, err.Error(), "run_once", "the error must name the conflicting setting")

	// A separate group is fine.
	_, releaseOther := mustAcquire(t, g, logger, "other", runOnce)
	releaseOther()
}

func TestAcquireDoesNotWarnAboutMatchingParams(t *testing.T) {
	g := newGroups()
	logger, observed := logptest.NewTestingLoggerWithObserver(t, "")

	_, releaseFirst := mustAcquire(t, g, logger, "group", testParams())
	defer releaseFirst()
	// The registry is only used by the consumer that creates the scheduler, so
	// a different one must not count as a conflict.
	_, releaseSecond := mustAcquire(t, g, logger, "group", testParams())
	defer releaseSecond()

	// Params of a different group are unrelated and must not be compared.
	otherGroup := testParams()
	otherGroup.Limit = 20
	_, releaseOther := mustAcquire(t, g, logger, "other", otherGroup)
	defer releaseOther()

	assert.Empty(t, observed.FilterLevelExact(logp.WarnLevel.ZapLevel()).All())
}

func TestAcquireIsConcurrencySafe(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	const consumers = 32
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		acquired []*scheduler.Scheduler
		releases []ReleaseFunc
	)

	wg.Add(consumers)
	for range consumers {
		go func() {
			defer wg.Done()
			sched, release := mustAcquire(t, g, logger, "group", testParams())

			mu.Lock()
			defer mu.Unlock()
			acquired = append(acquired, sched)
			releases = append(releases, release)
		}()
	}
	wg.Wait()

	require.Len(t, acquired, consumers)
	for _, sched := range acquired {
		assert.Same(t, acquired[0], sched)
	}

	wg.Add(consumers)
	for _, release := range releases {
		go func(release ReleaseFunc) {
			defer wg.Done()
			release()
		}(release)
	}
	wg.Wait()

	assert.True(t, stopped(t, acquired[0]), "the scheduler must stop once all consumers released it")
}

// TestJobLimitsAreSharedWithinGroup covers the point of this package, and so of
// https://github.com/elastic/elastic-agent/issues/16283: a per job type limit of
// one must let only a single job of a group run at a time, no matter how many
// consumers added jobs.
func TestJobLimitsAreSharedWithinGroup(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	params := testParams()
	params.JobLimitByType = map[string]*config.JobLimit{"browser": {Limit: 1}}

	first, releaseFirst := mustAcquire(t, g, logger, "group", params)
	defer releaseFirst()
	second, releaseSecond := mustAcquire(t, g, logger, "group", params)
	defer releaseSecond()

	started := make(chan string)
	unblock := make(chan struct{})
	defer close(unblock)

	blockingJob(t, first, "first", started, unblock)
	blockingJob(t, second, "second", started, unblock)

	var firstToRun string
	select {
	case firstToRun = <-started:
	case <-time.After(30 * time.Second):
		require.Fail(t, "no job started")
	}

	// While the first job holds the only browser slot the other one must wait,
	// even though it was added through a different consumer.
	select {
	case id := <-started:
		require.Failf(t, "browser limit was not shared", "job %q ran while %q held the only slot", id, firstToRun)
	case <-time.After(500 * time.Millisecond):
	}
}

// Different groups are independent, which is what keeps a standalone Heartbeat
// process, and a receiver that opted out of grouping, unaffected.
func TestJobLimitsAreNotSharedAcrossGroups(t *testing.T) {
	g := newGroups()
	logger := logptest.NewTestingLogger(t, "")

	params := testParams()
	params.JobLimitByType = map[string]*config.JobLimit{"browser": {Limit: 1}}

	first, releaseFirst := mustAcquire(t, g, logger, "first", params)
	defer releaseFirst()
	second, releaseSecond := mustAcquire(t, g, logger, "second", params)
	defer releaseSecond()

	started := make(chan string)
	unblock := make(chan struct{})
	defer close(unblock)

	blockingJob(t, first, "first", started, unblock)
	blockingJob(t, second, "second", started, unblock)

	ran := map[string]bool{}
	for range 2 {
		select {
		case id := <-started:
			ran[id] = true
		case <-time.After(30 * time.Second):
			require.Failf(t, "a job of another group was blocked", "only %v ran", ran)
		}
	}
	assert.Equal(t, map[string]bool{"first": true, "second": true}, ran)
}

// TestAcquire checks that Acquire shares schedulers through the process-wide
// groups. The behavior of groups is covered by the tests above.
func TestAcquire(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	// Use a group of its own, so other users of the process-wide groups in
	// this test binary don't interfere.
	group := t.Name()

	first, releaseFirst, err := Acquire(logger, group, testParams())
	require.NoError(t, err)
	second, releaseSecond, err := Acquire(logger, group, testParams())
	require.NoError(t, err)
	assert.Same(t, first, second, "consumers of a group must get the same scheduler")

	releaseFirst()
	releaseSecond()
	assert.True(t, stopped(t, first), "the scheduler must stop once its last consumer released it")
}
