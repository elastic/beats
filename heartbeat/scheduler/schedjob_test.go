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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestSchedJobRun(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := []struct {
		name          string
		jobCtx        context.Context
		overLimit     bool
		shouldRunTask bool
	}{
		{
			"context not cancelled",
			context.Background(),
			false,
			true,
		},
		{
			"context cancelled",
			cancelledCtx,
			false,
			false,
		},
		{
			"context cancelled over limit",
			cancelledCtx,
			true,
			false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			limit := int64(100)
			s := Create(limit, monitoring.NewRegistry(), tarawaTime(), nil, false, logptest.NewTestingLogger(t, ""))

			if testCase.overLimit {
				err := s.limitSem.Acquire(context.Background(), limit)
				require.NoError(t, err)
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)
			executed := &atomic.Bool{}

			tf := func(ctx context.Context) []TaskFunc {
				executed.Store(true)
				return nil
			}

			beforeStart := time.Now()
			sj := newSchedJob(testCase.jobCtx, s, "myid", "atype", tf, logptest.NewTestingLogger(t, ""))
			startedAt, taskStarted := sj.run()

			// This will panic in the case where we don't check s.limitSem.Acquire
			// for an error value and released an unacquired resource in scheduler.go.
			// In that case this will release one more resource than allowed causing
			// the panic.
			if testCase.overLimit {
				s.limitSem.Release(limit)
			}

			require.Equal(t, testCase.shouldRunTask, executed.Load())
			require.Equal(t, testCase.shouldRunTask, taskStarted,
				"task-start signal should match task body execution")
			require.True(t, startedAt.Equal(beforeStart) || startedAt.After(beforeStart))
		})
	}
}

// testRecursiveForkingJob tests that a schedJob that splits into multiple parallel pieces executes without error
func TestRecursiveForkingJob(t *testing.T) {
	s := Create(1000, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
		"atype": {Limit: 1},
	}, false, logptest.NewTestingLogger(t, ""))
	var ran atomic.Int64

	var terminalTf TaskFunc = func(ctx context.Context) []TaskFunc {
		ran.Add(1)
		return nil
	}
	var forkingTf TaskFunc = func(ctx context.Context) []TaskFunc {
		ran.Add(1)
		return []TaskFunc{
			terminalTf, terminalTf, terminalTf,
		}
	}

	sj := newSchedJob(context.Background(), s, "myid", "atype", forkingTf, logptest.NewTestingLogger(t, ""))

	sj.run()
	require.Equal(t, int64(4), ran.Load())

}

func TestJobTypePressure(t *testing.T) {
	t.Run("tracks running and waiting jobs", func(t *testing.T) {
		s := Create(100, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
			"browser": {Limit: 1},
		}, false, logptest.NewTestingLogger(t, ""))
		defer s.Stop()

		firstStarted := make(chan struct{})
		releaseFirst := make(chan struct{})
		firstDone := make(chan struct{})
		secondStarted := make(chan struct{})
		secondDone := make(chan struct{})

		first := newSchedJob(context.Background(), s, "first", "browser", func(context.Context) []TaskFunc {
			close(firstStarted)
			<-releaseFirst
			return nil
		}, logptest.NewTestingLogger(t, ""))
		second := newSchedJob(context.Background(), s, "second", "browser", func(context.Context) []TaskFunc {
			close(secondStarted)
			return nil
		}, logptest.NewTestingLogger(t, ""))

		go func() {
			first.run()
			close(firstDone)
		}()
		<-firstStarted

		go func() {
			second.run()
			close(secondDone)
		}()

		require.Eventually(t, func() bool {
			status := s.Status()
			return status.Jobs["browser"].Running == 1 &&
				status.Jobs["browser"].Waiting == 1
		}, time.Second, 10*time.Millisecond,
			"one browser job should run while the second waits")

		status := s.Status()
		assert.Equal(t, int64(1), status.Jobs["browser"].Running,
			"one browser job should hold the type semaphore")
		assert.Equal(t, int64(1), status.Jobs["browser"].Waiting,
			"the second browser job should wait for the type semaphore")

		close(releaseFirst)
		<-firstDone
		<-secondStarted
		<-secondDone

		status = s.Status()
		assert.Equal(t, int64(0), status.Jobs["browser"].Running,
			"completed browser jobs should not be reported as running")
		assert.Equal(t, int64(0), status.Jobs["browser"].Waiting,
			"completed browser jobs should not be reported as waiting")
		assert.Equal(t, uint64(2), status.Jobs["browser"].Runs,
			"both browser tasks should be counted as runs")
	})

	t.Run("does not count canceled jobs", func(t *testing.T) {
		s := Create(100, monitoring.NewRegistry(), tarawaTime(), map[string]*config.JobLimit{
			"browser": {Limit: 1},
		}, false, logptest.NewTestingLogger(t, ""))
		defer s.Stop()

		require.NoError(t, s.jobLimitSem["browser"].Acquire(context.Background(), 1),
			"browser semaphore should be available")
		defer s.jobLimitSem["browser"].Release(1)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		executed := false
		job := newSchedJob(ctx, s, "canceled", "browser", func(context.Context) []TaskFunc {
			executed = true
			return nil
		}, logptest.NewTestingLogger(t, ""))
		job.run()

		status := s.Status()
		assert.False(t, executed, "a canceled browser job should not execute")
		assert.Equal(t, int64(0), status.Jobs["browser"].Running,
			"a canceled browser job should not be reported as running")
		assert.Equal(t, int64(0), status.Jobs["browser"].Waiting,
			"a canceled browser job should not remain waiting")
		assert.Equal(t, uint64(0), status.Jobs["browser"].Runs,
			"a canceled browser job should not be counted as a run")
	})
}
