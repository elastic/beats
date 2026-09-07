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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/heartbeat/scheduler"
)

func TestSchedulerPayload(t *testing.T) {
	status := scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"browser": {
				Limit:   2,
				Running: 0,
				Waiting: 0,
				Runs:    0,
				ScheduleDelay: scheduler.ScheduleDelayStatus{
					Count:   0,
					TotalMS: 0,
					MaxMS:   0,
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
						"running": int64(0),
						"waiting": int64(0),
						"runs":    uint64(0),
						"schedule_delay": map[string]any{
							"count":    uint64(0),
							"total_ms": uint64(0),
							"max_ms":   uint64(0),
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

func TestSchedulerPayloadFiltersUnsupportedJobTypes(t *testing.T) {
	status := scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http":         {Limit: 10, Running: 1},
			"operator-key": {Limit: 99, Running: 98},
		},
	}

	got := schedulerPayload(status)
	jobs := got["heartbeat"].(map[string]any)["scheduler"].(map[string]any)["jobs"].(map[string]any)

	assert.Equal(t, map[string]any{
		"http": map[string]any{
			"limit":   int64(10),
			"running": int64(1),
			"waiting": int64(0),
			"runs":    uint64(0),
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
	payloads chan map[string]any
}

func (s *recordingPayloadSetter) SetPayload(payload map[string]any) {
	s.payloads <- payload
}

type mutableSchedulerStatus struct {
	status scheduler.Status
}

func (s *mutableSchedulerStatus) Status() scheduler.Status {
	return s.status
}

func TestSchedulerPayloadReporterSendsImmediatelyAndOnEachTick(t *testing.T) {
	setter := &recordingPayloadSetter{payloads: make(chan map[string]any, 3)}
	source := &mutableSchedulerStatus{
		status: scheduler.Status{
			Jobs: map[string]scheduler.JobTypeStatus{
				"http": {Limit: 10, Running: 1},
			},
		},
	}
	ticks := make(chan time.Time)

	stop := startSchedulerPayloadReporterWithTicks(setter, source, ticks)
	defer stop()

	assert.Equal(t, schedulerPayload(source.status), <-setter.payloads, "reporter should send an immediate snapshot")

	source.status = scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http": {Limit: 10, Running: 2, Waiting: 1},
		},
	}
	ticks <- time.Time{}
	assert.Equal(t, schedulerPayload(source.status), <-setter.payloads, "reporter should send the first tick snapshot")

	source.status = scheduler.Status{
		Jobs: map[string]scheduler.JobTypeStatus{
			"http": {Limit: 10, Running: 3, Runs: 4},
		},
	}
	ticks <- time.Time{}
	assert.Equal(t, schedulerPayload(source.status), <-setter.payloads, "reporter should send the second tick snapshot")
}

func TestSchedulerPayloadReporterStopsOnCancellation(t *testing.T) {
	setter := &recordingPayloadSetter{payloads: make(chan map[string]any, 2)}
	source := &mutableSchedulerStatus{status: scheduler.Status{Jobs: map[string]scheduler.JobTypeStatus{}}}
	ticks := make(chan time.Time, 1)

	stop := startSchedulerPayloadReporterWithTicks(setter, source, ticks)
	<-setter.payloads
	stop()

	ticks <- time.Time{}
	assert.Empty(t, setter.payloads, "reporter should not send snapshots after it stops")
}

func TestSchedulerPayloadReporterDoesNotStartWhenManagementDisabled(t *testing.T) {
	setter := &recordingPayloadSetter{payloads: make(chan map[string]any, 1)}
	source := &mutableSchedulerStatus{status: scheduler.Status{Jobs: map[string]scheduler.JobTypeStatus{}}}

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
	setter := &recordingPayloadSetter{payloads: make(chan map[string]any, 1)}
	source := &mutableSchedulerStatus{status: scheduler.Status{Jobs: map[string]scheduler.JobTypeStatus{}}}

	stop := heartbeat.startManagedSchedulerPayloadReporter(true, setter, source)
	defer stop()

	assert.Empty(t, setter.payloads, "stopped heartbeat should not start the reporter")
}
