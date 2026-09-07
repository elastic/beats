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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestSchedulerStatus(t *testing.T) {
	scheduler := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		"browser": {Limit: 2},
		"icmp":    {Limit: -1},
	}, false, logptest.NewTestingLogger(t, ""))
	defer scheduler.Stop()

	status := scheduler.Status()
	assert.Equal(t, int64(2), status.Jobs["browser"].Limit,
		"configured browser limit should be reported")
	assert.Equal(t, int64(0), status.Jobs["browser"].Running,
		"new scheduler should report no running browser jobs")
	assert.Equal(t, int64(0), status.Jobs["browser"].Waiting,
		"new scheduler should report no waiting browser jobs")
	assert.Equal(t, int64(0), status.Jobs["icmp"].Limit,
		"negative job limits should be reported as unlimited")

	scheduler.getJobTypeStats("http")
	status = scheduler.Status()
	assert.Equal(t, int64(0), status.Jobs["http"].Limit,
		"unknown job types should be reported as unlimited")
}

func TestSchedulerStatusNilJobLimit(t *testing.T) {
	scheduler := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		"browser": nil,
	}, false, logptest.NewTestingLogger(t, ""))
	defer scheduler.Stop()

	status := scheduler.Status()
	assert.Equal(t, int64(0), status.Jobs["browser"].Limit,
		"nil job limits should be reported as unlimited")
}

func TestJobTypeScheduleDelay(t *testing.T) {
	stats := &jobTypeStats{}
	stats.recordDelay(100 * time.Millisecond)
	stats.recordDelay(900 * time.Millisecond)
	stats.recordDelay(-time.Second)

	delay := ScheduleDelayStatus{
		Count:   stats.delayCount.Load(),
		TotalMS: stats.delayTotalMS.Load(),
		MaxMS:   stats.delayMaxMS.Load(),
	}
	assert.Equal(t, uint64(3), delay.Count,
		"all started jobs should contribute a delay sample")
	assert.Equal(t, uint64(1000), delay.TotalMS,
		"negative delay should be clamped to zero")
	assert.Equal(t, uint64(900), delay.MaxMS,
		"maximum delay should retain the largest sample")
}
