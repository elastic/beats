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

package memqueue

import (
	"context"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TestPublishPoolNoAllocsInSteadyState verifies the central claim of the
// sync.Pool optimization: in steady state, Publish allocates no new response
// channels.  Without the pool, each call to makePushRequest contained
// make(chan queue.EntryID, 1) — one heap allocation per publish.  With the
// pool, the same channel is acquired via pool.Get and returned via
// defer pool.Put, so the per-publish channel alloc drops to zero.
func TestPublishPoolNoAllocsInSteadyState(t *testing.T) {
	q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
		Events:        64,
		MaxGetRequest: 64,
		FlushTimeout:  time.Millisecond,
	}, 0, nil)
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{})
	defer p.Close()

	// Background consumer: keeps the queue drained so Publish never blocks
	// waiting for space and the pool.Put defer always fires promptly.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for ctx.Err() == nil {
			batch, err := q.Get(64)
			if err != nil {
				return
			}
			batch.Done()
		}
	}()

	// Warm up: a few publishes let the pool acquire its first channel before
	// we start measuring.
	for range 8 {
		_, ok := p.Publish(0)
		require.True(t, ok)
	}

	// Disable GC so the pool cannot be swept between iterations, making the
	// result deterministic.
	oldGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGC)

	// Each iteration must complete the full produce.go path:
	//   makePushRequest → getRespChan (pool.Get)
	//   openState.publish → defer pool.Put fires on return
	//   handlePendingResponse drains the channel before publish returns
	allocs := testing.AllocsPerRun(200, func() {
		_, _ = p.Publish(0)
	})

	// Without pool: 1 alloc/op (make(chan queue.EntryID, 1) in makePushRequest).
	// With pool warm: 0 allocs/op (channel comes from pool).
	assert.Equal(t, 0.0, allocs,
		"Publish should not allocate a response channel after pool warmup; "+
			"got %.1f allocs/op — check that pool.Put is reached after every publish", allocs)
}
