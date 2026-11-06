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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hbconfig "github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestOutputExpectedConfigJobLimitsOverrideStartupFallbacks(t *testing.T) {
	t.Setenv("SYNTHETICS_LIMIT_BROWSER", "3")
	t.Setenv("SYNTHETICS_LIMIT_API", "5")

	fallbacks := hbconfig.DefaultConfig(logptest.NewTestingLogger(t, ""))
	heartbeatConfig, err := conf.NewConfigFrom(map[string]any{
		"jobs": map[string]any{
			"browser": map[string]any{"limit": 2},
			"api":     map[string]any{"limit": 6},
		},
	})
	require.NoError(t, err, "heartbeat.yml fallback config should parse")
	require.NoError(t, heartbeatConfig.Unpack(fallbacks), "heartbeat.yml should override environment limits")

	s := scheduler.Create(10, monitoring.NewRegistry(), time.Local, fallbacks.Jobs, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()
	bt := &Heartbeat{scheduler: s, logger: logptest.NewTestingLogger(t, "")}

	outputConfig, err := conf.NewConfigFrom(map[string]any{
		"heartbeat": map[string]any{
			"jobs": map[string]any{
				"browser": map[string]any{"limit": 1},
			},
		},
	})
	require.NoError(t, err, "output expected config should parse")
	bt.applyOutputJobLimits(outputConfig)

	firstStarted, firstRelease, firstDone := startBlockingMonitor(t, s, "browser", "first")
	requireClosed(t, firstStarted, "first browser monitor should start")
	secondStarted, secondRelease, secondDone := startBlockingMonitor(t, s, "browser", "second")
	assertNotClosed(t, secondStarted, "Fleet browser limit should override heartbeat.yml and environment limits")

	apiOnlyConfig, err := conf.NewConfigFrom(map[string]any{
		"heartbeat": map[string]any{
			"jobs": map[string]any{
				"api": map[string]any{"limit": 7},
			},
		},
	})
	require.NoError(t, err, "partial output expected config should parse")
	bt.applyOutputJobLimits(apiOnlyConfig)
	requireClosed(t, secondStarted, "an omitted Fleet type should return to its heartbeat.yml fallback")

	close(firstRelease)
	close(secondRelease)
	requireClosed(t, firstDone, "first browser monitor should finish")
	requireClosed(t, secondDone, "second browser monitor should finish")
}

func TestOutputExpectedConfigInvalidJobsKeepLastGoodLimits(t *testing.T) {
	s := scheduler.Create(10, monitoring.NewRegistry(), time.Local, map[string]*hbconfig.JobLimit{
		"browser": {Limit: 2},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()
	bt := &Heartbeat{scheduler: s, logger: logptest.NewTestingLogger(t, "")}

	valid, err := conf.NewConfigFrom(map[string]any{
		"heartbeat": map[string]any{"jobs": map[string]any{"browser": map[string]any{"limit": 1}}},
	})
	require.NoError(t, err, "valid output expected config should parse")
	bt.applyOutputJobLimits(valid)

	firstStarted, firstRelease, firstDone := startBlockingMonitor(t, s, "browser", "first")
	requireClosed(t, firstStarted, "first browser monitor should start")
	secondStarted, secondRelease, secondDone := startBlockingMonitor(t, s, "browser", "second")
	assertNotClosed(t, secondStarted, "valid Fleet limit should constrain browser monitors")

	malformed, err := conf.NewConfigFrom(map[string]any{
		"heartbeat": map[string]any{"jobs": "not-a-map"},
	})
	require.NoError(t, err, "malformed field container should still parse")
	bt.applyOutputJobLimits(malformed)
	assertNotClosed(t, secondStarted, "malformed output config must retain the last good limit")

	missing, err := conf.NewConfigFrom(map[string]any{"hosts": []string{"https://example.invalid"}})
	require.NoError(t, err, "ordinary output config should parse")
	bt.applyOutputJobLimits(missing)
	assertNotClosed(t, secondStarted, "missing heartbeat.jobs must retain the last good limit")

	close(firstRelease)
	requireClosed(t, firstDone, "first browser monitor should finish")
	requireClosed(t, secondStarted, "waiting monitor should run when its slot is released")
	close(secondRelease)
	requireClosed(t, secondDone, "second browser monitor should finish")
}

type immediateSchedule struct{}

func (immediateSchedule) Next(now time.Time) time.Time { return now.Add(time.Hour) }
func (immediateSchedule) RunOnInit() bool              { return true }

func startBlockingMonitor(t *testing.T, s *scheduler.Scheduler, jobType, id string) (started, release, done chan struct{}) {
	t.Helper()
	started = make(chan struct{})
	release = make(chan struct{})
	done = make(chan struct{})
	_, err := s.Add(immediateSchedule{}, nil, id, func(context.Context) []scheduler.TaskFunc {
		close(started)
		<-release
		close(done)
		return nil
	}, jobType)
	require.NoError(t, err, "blocking monitor should register with the scheduler")
	return started, release, done
}

func assertNotClosed(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
		assert.Fail(t, message)
	case <-time.After(100 * time.Millisecond):
	}
}

func requireClosed(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		require.FailNow(t, message)
	}
}
