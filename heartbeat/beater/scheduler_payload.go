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

	"github.com/elastic/beats/v7/heartbeat/scheduler"
)

const schedulerPayloadInterval = 30 * time.Second

type payloadSetter interface {
	SetPayload(map[string]any)
}

type schedulerStatusProvider interface {
	Status() scheduler.Status
}

func schedulerPayload(status scheduler.Status) map[string]any {
	jobs := make(map[string]any, len(status.Jobs))
	for jobType, status := range status.Jobs {
		jobs[jobType] = map[string]any{
			"limit":   status.Limit,
			"running": status.Running,
			"waiting": status.Waiting,
			"runs":    status.Runs,
			"schedule_delay": map[string]any{
				"count":    status.ScheduleDelay.Count,
				"total_ms": status.ScheduleDelay.TotalMS,
				"max_ms":   status.ScheduleDelay.MaxMS,
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

func startSchedulerPayloadReporterWithTicks(
	setter payloadSetter,
	statusProvider schedulerStatusProvider,
	ticks <-chan time.Time,
) func() {
	setter.SetPayload(schedulerPayload(statusProvider.Status()))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticks:
				setter.SetPayload(schedulerPayload(statusProvider.Status()))
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
