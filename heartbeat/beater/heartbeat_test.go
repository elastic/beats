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

	"github.com/elastic/beats/v7/heartbeat/tracer"
	"github.com/elastic/beats/v7/libbeat/beat"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestMakeESClient(t *testing.T) {
	t.Run("should not modify the timeout setting from original config", func(t *testing.T) {
		origTimeout := 90
		origCfg, _ := conf.NewConfigFrom(map[any]any{
			"hosts":    []string{"http://localhost:9200"},
			"username": "anyuser",
			"password": "anypwd",
			"timeout":  origTimeout,
		})
		anyAttempt := 1
		anyDuration := 1 * time.Second

		_, _ = makeESClient(context.Background(), origCfg, anyAttempt, anyDuration, logptest.NewTestingLogger(t, ""), beat.Info{})

		timeout, err := origCfg.Int("timeout", -1)
		require.NoError(t, err)
		assert.EqualValues(t, origTimeout, timeout)
	})
}

func TestStopBeforeRunReleasesScheduler(t *testing.T) {
	released := 0
	bt := &Heartbeat{
		done:             make(chan struct{}),
		releaseScheduler: func() { released++ },
		trace:            tracer.NewNoopTracer(),
		logger:           logptest.NewTestingLogger(t, ""),
	}

	// A collector may create a receiver and give up on it without ever starting
	// it. The scheduler it acquired has to be released anyway, otherwise it is
	// pinned for the lifetime of the process.
	bt.Stop()
	assert.Equal(t, 1, released, "Stop must release the scheduler when Run never started")

	// Run has nothing left to do, and must not release a second time.
	require.NoError(t, bt.Run(nil))
	assert.Equal(t, 1, released)
}

func TestStopAfterRunLeavesReleaseToRun(t *testing.T) {
	released := 0
	bt := &Heartbeat{
		done:             make(chan struct{}),
		releaseScheduler: func() { released++ },
		logger:           logptest.NewTestingLogger(t, ""),
	}

	// Stand in for a Run that is under way: it releases the scheduler when it
	// returns, after its monitors have wound down, so Stop must not do it early.
	bt.schedMu.Lock()
	bt.runStarted = true
	bt.schedMu.Unlock()

	bt.Stop()
	assert.Zero(t, released, "Run is responsible for releasing the scheduler once it started")
}
