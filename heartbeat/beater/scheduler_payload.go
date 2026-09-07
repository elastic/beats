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
	"context"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
)

const schedulerPayloadInterval = 30 * time.Second

// payloadSetter attaches scheduler telemetry to the Elastic Agent output unit.
type payloadSetter interface {
	SetOutputPayload(map[string]any)
}

type schedulerStatusProvider interface {
	Status() scheduler.Status
}

// schedulerPayload renders the snapshot consumers read from the single Elastic
// Agent output unit (`unit.type == "output"`). Per monitor type it exposes
// exactly `limit`, `running`, `waiting` and
// `schedule_delay.{count,total_ms,max_ms}`.
//
// `waiting` counts jobs blocked on the per-type semaphore, so types without a
// per-type limit report 0; their global scheduler pressure shows up as
// schedule-delay events instead. `max_ms` is a process-lifetime maximum, so
// sustained pressure must be derived from deltas of `count` and `total_ms`
// combined with the current `waiting`, not from `max_ms` alone.
func schedulerPayload(status scheduler.Status) map[string]any {
	jobs := make(map[string]any, len(status.Jobs))
	for jobType, jobStatus := range status.Jobs {
		if !config.IsSupportedJobType(jobType) {
			continue
		}
		jobs[jobType] = map[string]any{
			"limit":   jobStatus.Limit,
			"running": jobStatus.Running,
			"waiting": jobStatus.Waiting,
			"schedule_delay": map[string]any{
				"count":    jobStatus.ScheduleDelay.Count,
				"total_ms": jobStatus.ScheduleDelay.TotalMS,
				"max_ms":   jobStatus.ScheduleDelay.MaxMS,
			},
		}
	}

	return map[string]any{
		"heartbeat": map[string]any{
			"scheduler": map[string]any{
				"jobs": jobs,
			},
		},
	}
}

func startManagedSchedulerPayloadReporter(
	managed bool,
	setter payloadSetter,
	statusProvider schedulerStatusProvider,
) func() {
	if !managed {
		return func() {}
	}

	ticker := time.NewTicker(schedulerPayloadInterval)
	stop := startSchedulerPayloadReporterWithTicks(setter, statusProvider, ticker.C)
	return func() {
		ticker.Stop()
		stop()
	}
}

func (bt *Heartbeat) startManagedSchedulerPayloadReporter(
	managed bool,
	setter payloadSetter,
	statusProvider schedulerStatusProvider,
) func() {
	bt.schedulerPayloadReporterMu.Lock()
	defer bt.schedulerPayloadReporterMu.Unlock()

	select {
	case <-bt.done:
		return func() {}
	default:
	}

	stop := startManagedSchedulerPayloadReporter(managed, setter, statusProvider)
	bt.schedulerPayloadReporterStop = stop
	return stop
}

// startSchedulerPayloadReporterWithTicks publishes a snapshot immediately and
// then on every tick. Snapshots are unconditionally handed to the manager,
// which drops values identical to the last one it forwarded; a scheduler under
// no pressure therefore produces no periodic Fleet state writes.
func startSchedulerPayloadReporterWithTicks(
	setter payloadSetter,
	statusProvider schedulerStatusProvider,
	ticks <-chan time.Time,
) func() {
	setter.SetOutputPayload(schedulerPayload(statusProvider.Status()))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticks:
				setter.SetOutputPayload(schedulerPayload(statusProvider.Status()))
			}
		}
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			cancel()
			<-done
		})
	}
}
