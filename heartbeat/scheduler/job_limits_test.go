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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestApplyJobLimitsTransitionsBetweenLimitedAndUnlimited(t *testing.T) {
	const jobType = "http"
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		jobType: {Limit: 0},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	require.NoError(t, s.ApplyJobLimits(map[string]*config.JobLimit{
		jobType: {Limit: 1},
	}), "Fleet should be able to add a limit to an unlimited type")

	firstStarted, firstRelease, firstDone := startBlockingJob(t, s, jobType, "first")
	requireClosed(t, firstStarted, "first limited job should start")
	secondStarted, secondRelease, secondDone := startBlockingJob(t, s, jobType, "second")
	assertNotClosed(t, secondStarted, "second job should wait for the Fleet limit")

	require.NoError(t, s.ApplyJobLimits(map[string]*config.JobLimit{
		jobType: {Limit: 0},
	}), "Fleet should be able to remove a type limit")
	requireClosed(t, secondStarted, "removing the limit should release the waiting job")

	close(firstRelease)
	close(secondRelease)
	requireClosed(t, firstDone, "first job should finish")
	requireClosed(t, secondDone, "second job should finish")
}

func TestApplyJobLimitsDrainsWhenReducingConcurrency(t *testing.T) {
	const jobType = "browser"
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		jobType: {Limit: 3},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	started := make([]chan struct{}, 3)
	release := make([]chan struct{}, 3)
	done := make([]chan struct{}, 3)
	for i := range started {
		started[i], release[i], done[i] = startBlockingJob(t, s, jobType, "initial")
		requireClosed(t, started[i], "initial jobs should all start before the cap is reduced")
	}

	require.NoError(t, s.ApplyJobLimits(map[string]*config.JobLimit{
		jobType: {Limit: 1},
	}), "Fleet should be able to reduce the cap")

	fourthStarted, fourthRelease, fourthDone := startBlockingJob(t, s, jobType, "fourth")
	assertNotClosed(t, fourthStarted, "new work should wait while the old cap drains")

	for i := 0; i < 2; i++ {
		close(release[i])
		requireClosed(t, done[i], "released in-flight job should finish")
		assertNotClosed(t, fourthStarted, "new work must not start above the reduced cap")
	}

	close(release[2])
	requireClosed(t, done[2], "last over-subscribed job should finish")
	requireClosed(t, fourthStarted, "new work should start after the scheduler drains to the cap")
	close(fourthRelease)
	requireClosed(t, fourthDone, "waiting job should finish without deadlocking")
}

func TestApplyJobLimitsRestoresStartupFallbacksForOmittedTypes(t *testing.T) {
	s := Create(10, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		"browser": {Limit: 2},
		"api":     {Limit: 4},
	}, false, logptest.NewTestingLogger(t, ""))
	defer s.Stop()

	require.NoError(t, s.ApplyJobLimits(map[string]*config.JobLimit{
		"browser": {Limit: 1},
	}), "Fleet browser limit should apply")
	assert.Equal(t, int64(1), jobLimit(t, s, "browser"), "Fleet limit should take precedence over the startup fallback")

	require.NoError(t, s.ApplyJobLimits(map[string]*config.JobLimit{
		"api": {Limit: 5},
	}), "Fleet API limit should apply")
	assert.Equal(t, int64(2), jobLimit(t, s, "browser"), "an omitted Fleet type should return to its startup fallback")
	assert.Equal(t, int64(5), jobLimit(t, s, "api"), "the provided Fleet type should use its new limit")
}

func startBlockingJob(t *testing.T, s *Scheduler, jobType, id string) (started, release, done chan struct{}) {
	started = make(chan struct{})
	release = make(chan struct{})
	done = make(chan struct{})
	go func() {
		_ = newSchedJob(context.Background(), s, id, jobType, func(context.Context) []TaskFunc {
			close(started)
			<-release
			return nil
		}, logptest.NewTestingLogger(t, "")).run()
		close(done)
	}()
	return started, release, done
}

func jobLimit(t *testing.T, s *Scheduler, jobType string) int64 {
	t.Helper()
	sem := s.getJobLimitSem(jobType)
	sem.mu.Lock()
	defer sem.mu.Unlock()
	return sem.limit
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
