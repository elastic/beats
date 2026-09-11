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

package beater

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/heartbeat/scheduler"
)

// testTimeout bounds every channel wait so a regression fails the test instead
// of hanging the package.
const testTimeout = 10 * time.Second

func TestSchedulerPayload(t *testing.T) {
	status := scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"browser": {
				Limit:   2,
				Running: 1,
				Waiting: 3,
				ScheduleDelay: scheduler.ScheduleDelayStatus{
					Count:   2,
					TotalMS: 4500,
					MaxMS:   3000,
				},
			},
		},
	}

	got := schedulerPayload(status)

	expected := map[string]any{
		"heartbeat": map[string]any{
			"scheduler": map[string]any{
				"jobs": map[string]any{
					"browser": map[string]any{
						"limit":   int64(2),
						"running": int64(1),
						"waiting": int64(3),
						"schedule_delay": map[string]any{
							"count":    uint64(2),
							"total_ms": uint64(4500),
							"max_ms":   uint64(3000),
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, got, "scheduler payload should have the managed unit shape")

	_, err := structpb.NewStruct(got)
	assert.NoError(t, err, "scheduler payload should be protobuf-serializable")
}

func TestSchedulerPayloadHasExactlyTheContractKeys(t *testing.T) {
	got := schedulerPayload(scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{"http": {}},
	})

	http := payloadJob(t, got, "http")

	assert.ElementsMatch(t, []string{"limit", "running", "waiting", "schedule_delay"}, keysOf(http),
		"per-type payload must expose exactly the documented gauges")
	assert.ElementsMatch(t, []string{"count", "total_ms", "max_ms"}, keysOf(asMap(t, http["schedule_delay"], "schedule_delay")),
		"schedule_delay must expose exactly the documented delayed-start counters")
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func asMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	m, ok := value.(map[string]any)
	require.True(t, ok, "%s should be a map", name)
	return m
}

func payloadJobs(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	heartbeat := asMap(t, payload["heartbeat"], "heartbeat")
	sched := asMap(t, heartbeat["scheduler"], "scheduler")
	return asMap(t, sched["jobs"], "jobs")
}

func payloadJob(t *testing.T, payload map[string]any, jobType string) map[string]any {
	t.Helper()
	return asMap(t, payloadJobs(t, payload)[jobType], jobType)
}

func TestSchedulerPayloadFiltersUnsupportedJobTypes(t *testing.T) {
	status := scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http":         {Limit: 10, Running: 1},
			"operator-key": {Limit: 99, Running: 98},
		},
	}

	got := schedulerPayload(status)
	jobs := payloadJobs(t, got)

	assert.Equal(t, map[string]any{
		"http": map[string]any{
			"limit":   int64(10),
			"running": int64(1),
			"waiting": int64(0),
			"schedule_delay": map[string]any{
				"count":    uint64(0),
				"total_ms": uint64(0),
				"max_ms":   uint64(0),
			},
		},
	}, jobs, "scheduler payload should contain only supported monitor types")

	_, err := structpb.NewStruct(got)
	assert.NoError(t, err, "filtered scheduler payload should be protobuf-serializable")
}

type recordingPayloadSetter struct {
	t        *testing.T
	payloads chan map[string]any
}

func newRecordingPayloadSetter(t *testing.T) *recordingPayloadSetter {
	return &recordingPayloadSetter{t: t, payloads: make(chan map[string]any, 8)}
}

// SetOutputPayload records the snapshot without blocking the reporter: a full
// buffer means the reporter published more snapshots than the test expects.
func (s *recordingPayloadSetter) SetOutputPayload(payload map[string]any) {
	select {
	case s.payloads <- payload:
	default:
		assert.Fail(s.t, "reporter published more payloads than expected")
	}
}

func (s *recordingPayloadSetter) next() (map[string]any, bool) {
	select {
	case payload := <-s.payloads:
		return payload, true
	case <-time.After(testTimeout):
		return nil, false
	}
}

func (s *recordingPayloadSetter) requireNext(t *testing.T, msg string) map[string]any {
	t.Helper()

	payload, ok := s.next()
	require.True(t, ok, msg)
	return payload
}

type mutableSchedulerStatus struct {
	mu     sync.Mutex
	status scheduler.Status
}

func (s *mutableSchedulerStatus) set(status scheduler.Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

func (s *mutableSchedulerStatus) get() scheduler.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *mutableSchedulerStatus) Status() scheduler.Status {
	return s.get()
}

func requireTick(t *testing.T, ticks chan time.Time) {
	t.Helper()

	select {
	case ticks <- time.Time{}:
	case <-time.After(testTimeout):
		require.FailNow(t, "reporter should consume scheduler payload ticks")
	}
}

func TestSchedulerPayloadReporterSendsImmediatelyAndOnEachTick(t *testing.T) {
	setter := newRecordingPayloadSetter(t)
	source := &mutableSchedulerStatus{}
	source.set(scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http": {Limit: 10, Running: 1},
		},
	})
	ticks := make(chan time.Time)

	stop := startSchedulerPayloadReporterWithTicks(setter, source, ticks)
	defer stop()

	assert.Equal(t, schedulerPayload(source.get()),
		setter.requireNext(t, "reporter should send an immediate snapshot"),
		"immediate snapshot should match the scheduler status")

	source.set(scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http": {Limit: 10, Running: 2, Waiting: 1},
		},
	})
	requireTick(t, ticks)
	assert.Equal(t, schedulerPayload(source.get()),
		setter.requireNext(t, "reporter should send the first tick snapshot"),
		"first tick snapshot should match the scheduler status")

	source.set(scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http": {Limit: 10, Running: 3, ScheduleDelay: scheduler.ScheduleDelayStatus{Count: 1, TotalMS: 1200, MaxMS: 1200}},
		},
	})
	requireTick(t, ticks)
	assert.Equal(t, schedulerPayload(source.get()),
		setter.requireNext(t, "reporter should send the second tick snapshot"),
		"second tick snapshot should match the scheduler status")
}

func TestSchedulerPayloadReporterStopsOnCancellation(t *testing.T) {
	setter := newRecordingPayloadSetter(t)
	source := &mutableSchedulerStatus{}
	source.set(scheduler.Status{Jobs: map[string]scheduler.JobTypeStatus{}})
	ticks := make(chan time.Time, 1)

	stop := startSchedulerPayloadReporterWithTicks(setter, source, ticks)
	setter.requireNext(t, "reporter should send an immediate snapshot")
	stop()

	requireTick(t, ticks)
	assert.Empty(t, setter.payloads, "reporter should not send snapshots after it stops")
}

func TestSchedulerPayloadReporterDoesNotStartWhenManagementDisabled(t *testing.T) {
	setter := newRecordingPayloadSetter(t)
	source := &mutableSchedulerStatus{}
	source.set(scheduler.Status{Jobs: map[string]scheduler.JobTypeStatus{}})

	stop := startManagedSchedulerPayloadReporter(false, setter, source)
	defer stop()

	assert.Empty(t, setter.payloads, "standalone mode should not send scheduler payloads")
}

func TestSchedulerPayloadReporterStopsBeforeHeartbeatShutdown(t *testing.T) {
	heartbeat := &Heartbeat{done: make(chan struct{})}
	reporterStopped := false
	heartbeat.schedulerPayloadReporterStop = func() {
		select {
		case <-heartbeat.done:
			assert.Fail(t, "heartbeat should still be running when the reporter stops")
		default:
		}
		reporterStopped = true
	}

	heartbeat.Stop()

	assert.True(t, reporterStopped, "heartbeat shutdown should stop the reporter")
	select {
	case <-heartbeat.done:
	default:
		assert.Fail(t, "heartbeat shutdown should signal completion")
	}
}

func TestSchedulerPayloadReporterDoesNotStartAfterHeartbeatStops(t *testing.T) {
	heartbeat := &Heartbeat{done: make(chan struct{})}
	heartbeat.Stop()
	setter := newRecordingPayloadSetter(t)
	source := &mutableSchedulerStatus{}
	source.set(scheduler.Status{Jobs: map[string]scheduler.JobTypeStatus{}})

	stop := heartbeat.startManagedSchedulerPayloadReporter(true, setter, source)
	defer stop()

	assert.Empty(t, setter.payloads, "stopped heartbeat should not start the reporter")
}
