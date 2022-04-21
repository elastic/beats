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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	batomic "github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/monitoring"
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
			s := Create(limit, monitoring.NewRegistry(), tarawaTime(), nil, false)

			if testCase.overLimit {
				s.limitSem.Acquire(context.Background(), limit)
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)
			executed := batomic.MakeBool(false)

			tf := func(ctx context.Context) []TaskFunc {
				executed.Store(true)
				return nil
			}

			beforeStart := time.Now()
			sj := newSchedJob(testCase.jobCtx, s, "myid", "atype", tf)
			startedAt := sj.run()

			// This will panic in the case where we don't check s.limitSem.Acquire
			// for an error value and released an unacquired resource in scheduler.go.
			// In that case this will release one more resource than allowed causing
			// the panic.
			if testCase.overLimit {
				s.limitSem.Release(limit)
			}

			require.Equal(t, testCase.shouldRunTask, executed.Load())
			require.True(t, startedAt.Equal(beforeStart) || startedAt.After(beforeStart))
		})
	}
}

// testRecursiveForkingJob tests that a schedJob that splits into multiple parallel pieces executes without error
func TestRecursiveForkingJob(t *testing.T) {
	s := Create(1000, monitoring.NewRegistry(), tarawaTime(), map[string]config.JobLimit{
		"atype": {Limit: 1},
	}, false)
	ran := batomic.NewInt(0)

	var terminalTf TaskFunc = func(ctx context.Context) []TaskFunc {
		ran.Inc()
		return nil
	}
	var forkingTf TaskFunc = func(ctx context.Context) []TaskFunc {
		ran.Inc()
		return []TaskFunc{
			terminalTf, terminalTf, terminalTf,
		}
	}

	sj := newSchedJob(context.Background(), s, "myid", "atype", forkingTf)

	sj.run()
	require.Equal(t, 4, ran.Load())

}
