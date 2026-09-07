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

package scheduler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/maintwin"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// The runAt in the island of tarawa 🏝. Good test TZ because it's pretty rare for a local box
// to be state in this TZ, and it has a weird offset +0125+17300.
func tarawaTime() *time.Location {
	loc, err := time.LoadLocation("Pacific/Tarawa")
	if err != nil {
		panic("this computer doesn't know about tarawa runAt " + err.Error())
	}

	return loc
}

func TestNewWithLocation(t *testing.T) {
	scheduler := Create(123, monitoring.NewRegistry(), tarawaTime(), nil, false, logptest.NewTestingLogger(t, ""))
	assert.Equal(t, int64(123), scheduler.limit)
	assert.Equal(t, tarawaTime(), scheduler.location)
}

// Runs tasks as fast as possible. Good for keeping tests snappy.
type testSchedule struct {
	delay time.Duration
}

func (testSchedule) RunOnInit() bool {
	return true
}

func (t testSchedule) Next(now time.Time) time.Time {
	return now.Add(t.delay)
}

type scheduledOnce struct {
	runAt time.Time
	calls atomic.Uint32
}

func (*scheduledOnce) RunOnInit() bool {
	return false
}

func (s *scheduledOnce) Next(time.Time) time.Time {
	if s.calls.Add(1) == 1 {
		return s.runAt
	}
	return s.runAt.Add(time.Hour)
}

// Test task that will only actually invoke the fn the given number of times
// this lets us test around timing / scheduling weirdness more accurately, since
// we can in tests expect an exact number of invocations
func testTaskTimes(limit uint32, fn TaskFunc) TaskFunc {
	invoked := new(uint32)
	return func(ctx context.Context) (conts []TaskFunc) {
		if atomic.LoadUint32(invoked) < limit {
			conts = fn(ctx)
		}
		atomic.AddUint32(invoked, 1)
		return conts
	}
}

func TestSchedulerRun(t *testing.T) {
	// We use tarawa runAt because it could expose some weird runAt math if by accident some code
	// relied on the local TZ.
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	mainWin := maintwin.ParsedMaintWin{}
	mainWins := []maintwin.ParsedMaintWin{mainWin}

	executed := make(chan string)

	initialEvents := uint32(10)
	_, err := s.Add(testSchedule{0}, mainWins, "add", testTaskTimes(initialEvents, func(_ context.Context) []TaskFunc {
		executed <- "initial"
		cont := func(_ context.Context) []TaskFunc {
			executed <- "initialCont"
			return nil
		}
		return []TaskFunc{cont}
	}), "http")
	require.NoError(t, err)

	removedEvents := uint32(1)
	// This function will be removed after being invoked once
	removeMtx := sync.Mutex{}
	var remove context.CancelFunc
	var testFn TaskFunc = func(_ context.Context) []TaskFunc {
		executed <- "removed"
		removeMtx.Lock()
		remove()
		removeMtx.Unlock()
		return nil
	}
	// Attempt to execute this twice to see if remove() had any effect
	removeMtx.Lock()
	remove, err = s.Add(testSchedule{}, mainWins, "removed", testTaskTimes(removedEvents+1, testFn), "http")
	require.NoError(t, err)
	require.NotNil(t, remove)
	removeMtx.Unlock()

	postRemoveEvents := uint32(10)
	_, err = s.Add(testSchedule{}, mainWins, "postRemove", testTaskTimes(postRemoveEvents, func(_ context.Context) []TaskFunc {
		executed <- "postRemove"
		cont := func(_ context.Context) []TaskFunc {
			executed <- "postRemoveCont"
			return nil
		}
		return []TaskFunc{cont}
	}), "http")
	require.NoError(t, err)

	received := make([]string, 0)
	// We test for a good number of events in this loop because we want to ensure that the remove() took effect
	// Otherwise, we might only do 1 preAdd and 1 postRemove event
	// We double the number of pre/post add events to account for their continuations
	totalExpected := initialEvents*2 + removedEvents + postRemoveEvents*2
	for uint32(len(received)) < totalExpected {
		select {
		case got := <-executed:
			received = append(received, got)
		case <-time.After(5 * time.Second):
			require.Fail(t, fmt.Sprintf("Timed out waitingTasks for schedule job to execute, got %d of %d: %v",
				len(received), totalExpected, received))
		}
	}

	// The removed callback should only have been executed once
	counts := map[string]uint32{"initial": 0, "initialCont": 0, "removed": 0, "postRemove": 0, "postRemoveCont": 0}
	for _, s := range received {
		counts[s]++
	}

	// convert with int() because the printed output is nicer than hex
	assert.Equal(t, int(initialEvents), int(counts["initial"]))
	assert.Equal(t, int(initialEvents), int(counts["initialCont"]))
	assert.Equal(t, int(removedEvents), int(counts["removed"]))
	assert.Equal(t, int(postRemoveEvents), int(counts["postRemove"]))
	assert.Equal(t, int(postRemoveEvents), int(counts["postRemoveCont"]))
}

func TestScheduleDelayIncludesTypeLimitWait(t *testing.T) {
	const jobType = "browser"
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		jobType: {Limit: 1},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan struct{})
	go func() {
		newSchedJob(context.Background(), s, "first", jobType, func(context.Context) []TaskFunc {
			close(firstStarted)
			<-releaseFirst
			return nil
		}, logptest.NewTestingLogger(t, "")).run()
		close(firstDone)
	}()
	requireClosed(t, firstStarted, "first job should start before scheduling the second")

	secondStarted := make(chan struct{})
	_, err := s.Add(&scheduledOnce{runAt: time.Now()}, nil, "second", func(context.Context) []TaskFunc {
		close(secondStarted)
		return nil
	}, jobType)
	require.NoError(t, err, "second job should be added")

	require.Eventually(t, func() bool {
		return s.Status().Jobs[jobType].Waiting == 1
	}, testTimeout, 10*time.Millisecond,
		"second job should block on the type semaphore")

	// Hold the only type slot past the reporting threshold so the recorded
	// delay is deterministically at or above it.
	holdFor := scheduleDelayThreshold + 200*time.Millisecond
	<-time.After(holdFor)
	close(releaseFirst)

	requireClosed(t, secondStarted, "second job should start after the first releases its type slot")
	requireClosed(t, firstDone, "first job should finish after release")

	require.Eventually(t, func() bool {
		return s.Status().Jobs[jobType].ScheduleDelay.Count == 1
	}, testTimeout, 10*time.Millisecond,
		"a start delayed past the threshold should be recorded once")
	delay := s.Status().Jobs[jobType].ScheduleDelay
	assert.GreaterOrEqual(t, delay.MaxMS, uint64(scheduleDelayThreshold.Milliseconds()),
		"second job delay should include time waiting for the type slot")
	assert.GreaterOrEqual(t, delay.TotalMS, uint64(scheduleDelayThreshold.Milliseconds()),
		"total delay should include time waiting for the type slot")
}

// TestUnlimitedJobTypeReportsPressureAsScheduleDelay pins the intentional
// asymmetry of the telemetry: `waiting` only tracks the per-type semaphore, so
// types without a per-type limit report 0 even while blocked on the global
// scheduler limit. Their pressure surfaces as a schedule-delay event instead.
func TestUnlimitedJobTypeReportsPressureAsScheduleDelay(t *testing.T) {
	const jobType = "http"
	s := Create(1, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		jobType: {Limit: 0},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	require.NoError(t, s.limitSem.Acquire(context.Background(), 1),
		"the only global scheduler slot should be free")

	started := make(chan struct{})
	_, err := s.Add(&scheduledOnce{runAt: time.Now().Add(-2 * scheduleDelayThreshold)}, nil, "blocked",
		func(context.Context) []TaskFunc {
			close(started)
			return nil
		}, jobType)
	require.NoError(t, err, "job should be added")

	require.Eventually(t, func() bool {
		return s.stats.waitingTasks.Get() == 1
	}, testTimeout, 10*time.Millisecond,
		"the job should block on the global scheduler limit")
	assert.Equal(t, int64(0), s.Status().Jobs[jobType].Waiting,
		"types without a per-type limit must report no waiting jobs")

	s.limitSem.Release(1)
	requireClosed(t, started, "job should run once a global slot frees")

	require.Eventually(t, func() bool {
		return s.Status().Jobs[jobType].ScheduleDelay.Count == 1
	}, testTimeout, 10*time.Millisecond,
		"global scheduler pressure should surface as a per-type schedule-delay event")
}

func TestOnTimeStartDoesNotRecordScheduleDelay(t *testing.T) {
	const jobType = "http"
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, true, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	executed := make(chan struct{}, 1)
	_, err := s.Add(testSchedule{}, nil, "ontime", func(context.Context) []TaskFunc {
		executed <- struct{}{}
		return nil
	}, jobType)
	require.NoError(t, err, "job should be added")
	s.WaitForRunOnce()

	select {
	case <-executed:
	case <-time.After(testTimeout):
		require.FailNow(t, "job should execute")
	}

	delay := s.Status().Jobs[jobType].ScheduleDelay
	assert.Equal(t, ScheduleDelayStatus{}, delay,
		"a start that was not late by at least the threshold must leave the counters untouched")
}

func TestCanceledJobDoesNotRecordScheduleDelay(t *testing.T) {
	const jobType = "browser"
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		jobType: {Limit: 1},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan struct{})
	go func() {
		newSchedJob(context.Background(), s, "first", jobType, func(context.Context) []TaskFunc {
			close(firstStarted)
			<-releaseFirst
			return nil
		}, logptest.NewTestingLogger(t, "")).run()
		close(firstDone)
	}()
	requireClosed(t, firstStarted, "first job should start before scheduling the canceled job")

	secondExecuted := make(chan struct{}, 1)
	removeSecond, err := s.Add(testSchedule{delay: time.Hour}, nil, "second", func(context.Context) []TaskFunc {
		secondExecuted <- struct{}{}
		return nil
	}, jobType)
	require.NoError(t, err, "second job should be added")
	require.Eventually(t, func() bool {
		return s.Status().Jobs[jobType].Waiting == 1
	}, testTimeout, 10*time.Millisecond,
		"second job should wait for the type slot")

	removeSecond()
	require.Eventually(t, func() bool {
		return s.Status().Jobs[jobType].Waiting == 0 && s.stats.activeJobs.Get() == 0
	}, testTimeout, 10*time.Millisecond,
		"canceled job should stop waiting for the type slot")
	close(releaseFirst)
	requireClosed(t, firstDone, "first job should finish after release")

	select {
	case <-secondExecuted:
		assert.Fail(t, "canceled job should not execute")
	default:
	}
	assert.Equal(t, uint64(0), s.Status().Jobs[jobType].ScheduleDelay.Count,
		"canceled job should not record schedule delay")
}

func TestStartedCanceledJobRecordsScheduleDelay(t *testing.T) {
	const jobType = "browser"
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	taskEntered := make(chan struct{})
	releaseTask := make(chan struct{})
	// Schedule in the past so the recorded delay clears the reporting
	// threshold without depending on wall-clock timing during the test.
	scheduledAt := time.Now().Add(-2 * scheduleDelayThreshold)
	remove, err := s.Add(&scheduledOnce{runAt: scheduledAt}, nil, "started", func(context.Context) []TaskFunc {
		close(taskEntered)
		<-releaseTask
		return nil
	}, jobType)
	require.NoError(t, err, "job should be added")

	requireClosed(t, taskEntered, "scheduled job should enter its task body")
	remove()
	close(releaseTask)
	require.Eventually(t, func() bool {
		return s.stats.activeJobs.Get() == 0
	}, testTimeout, 10*time.Millisecond,
		"started job should finish after release")

	delay := s.Status().Jobs[jobType].ScheduleDelay
	assert.Equal(t, uint64(1), delay.Count,
		"job canceled after its task body starts should record schedule delay")
	assert.GreaterOrEqual(t, delay.MaxMS, uint64(2*scheduleDelayThreshold.Milliseconds()),
		"recorded delay should reflect how late the start actually was")
}

func TestMaintenanceWindowSkipDoesNotRecordScheduleDelay(t *testing.T) {
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, true, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	window := maintwin.MaintWin{
		Freq:     "daily",
		Dtstart:  time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		Duration: 2 * time.Hour,
	}
	rule, err := window.Parse()
	require.NoError(t, err, "maintenance window should parse")

	executed := make(chan struct{}, 1)
	_, err = s.Add(testSchedule{}, []maintwin.ParsedMaintWin{{
		Rule:     rule,
		Duration: window.Duration,
	}}, "skipped", func(context.Context) []TaskFunc {
		executed <- struct{}{}
		return nil
	}, "http")
	require.NoError(t, err, "maintenance-window job should be added")
	s.WaitForRunOnce()

	select {
	case <-executed:
		require.Fail(t, "maintenance-window job should not execute")
	default:
	}
	assert.Equal(t, uint64(0), s.Status().Jobs["http"].ScheduleDelay.Count,
		"maintenance-window skip should not record schedule delay")
}

func TestScheduler_WaitForRunOnce(t *testing.T) {
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, true, logptest.NewTestingLogger(t, ""))

	defer s.Stop()

	mainWin := maintwin.ParsedMaintWin{}
	mainWins := []maintwin.ParsedMaintWin{mainWin}

	executed := new(uint32)

	_, err := s.Add(testSchedule{0}, mainWins, "runOnce", func(_ context.Context) []TaskFunc {
		cont := func(_ context.Context) []TaskFunc {
			// Make sure we actually wait for the task!
			time.Sleep(time.Millisecond * 250)
			atomic.AddUint32(executed, 1)
			return nil
		}
		return []TaskFunc{cont}
	}, "http")
	require.NoError(t, err)

	s.WaitForRunOnce()
	require.Equal(t, uint32(1), atomic.LoadUint32(executed))
}

func TestScheduler_Stop(t *testing.T) {
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), nil, false, logptest.NewTestingLogger(t, ""))

	executed := make(chan struct{})
	mainWin := maintwin.ParsedMaintWin{}
	mainWins := []maintwin.ParsedMaintWin{mainWin}

	s.Stop()

	_, err := s.Add(testSchedule{}, mainWins, "testPostStop", testTaskTimes(1, func(_ context.Context) []TaskFunc {
		executed <- struct{}{}
		return nil
	}), "http")

	assert.Equal(t, ErrAlreadyStopped, err)
}

func makeTasks(num int, callback func()) TaskFunc {
	return func(ctx context.Context) []TaskFunc {
		callback()
		if num < 1 {
			return nil
		}
		return []TaskFunc{makeTasks(num-1, callback)}
	}
}

func TestSchedTaskLimits(t *testing.T) {
	tests := []struct {
		name    string
		numJobs int
		limit   int64
		expect  func(events []int)
	}{
		{
			name:    "runs more than 1 with limit of 1",
			numJobs: 2,
			limit:   1,
			expect: func(events []int) {
				mid := len(events) / 2
				firstHalf := events[0:mid]
				lastHalf := events[mid:]
				for _, ele := range firstHalf {
					assert.Equal(t, firstHalf[0], ele)
				}
				for _, ele := range lastHalf {
					assert.Equal(t, lastHalf[0], ele)
				}
			},
		},
		{
			name:    "runs 50 interleaved without limit",
			numJobs: 50,
			limit:   math.MaxInt64,
			expect: func(events []int) {
				require.GreaterOrEqual(t, len(events), 250)
			},
		},
		{
			name:    "runs 100 with limit not configured",
			numJobs: 100,
			limit:   0,
			expect: func(events []int) {
				require.GreaterOrEqual(t, len(events), 500)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jobConfigByType = map[string]*config.JobLimit{}
			jobType := "http"
			if tt.limit > 0 {
				jobConfigByType = map[string]*config.JobLimit{
					jobType: {Limit: tt.limit},
				}
			}
			s := Create(math.MaxInt64, monitoring.NewRegistry(), tarawaTime(), jobConfigByType, false, logptest.NewTestingLogger(t, ""))
			var taskArr []int
			mtx := sync.Mutex{}
			wg := sync.WaitGroup{}
			wg.Add(tt.numJobs)
			for i := 0; i < tt.numJobs; i++ {
				num := i
				tf := makeTasks(4, func() {
					mtx.Lock()
					defer mtx.Unlock() // taskArr is shared across goroutines

					taskArr = append(taskArr, num)
				})
				go func(tff TaskFunc) {
					sj := newSchedJob(context.Background(), s, "myid", jobType, tff, logptest.NewTestingLogger(t, ""))
					sj.run()
					wg.Done()
				}(tf)
			}
			wg.Wait()
			tt.expect(taskArr)
		})
	}
}

func BenchmarkScheduler(b *testing.B) {
	s := Create(0, monitoring.NewRegistry(), tarawaTime(), nil, false, logptest.NewTestingLogger(b, ""))

	sched := testSchedule{0}
	mainWin := maintwin.ParsedMaintWin{}
	mainWins := []maintwin.ParsedMaintWin{mainWin}

	executed := make(chan struct{})
	for range 1024 {
		_, err := s.Add(sched, mainWins, "testPostStop", func(_ context.Context) []TaskFunc {
			executed <- struct{}{}
			return nil
		}, "http")
		assert.NoError(b, err)
	}

	defer s.Stop()

	count := 0
	for count < b.N {
		<-executed
		count++
	}
}
