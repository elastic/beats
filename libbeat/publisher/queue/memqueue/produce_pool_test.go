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
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TestRespChanAlwaysEmptyOnAcquire verifies the core safety invariant of the
// sync.Pool: a response channel returned to the pool by a completed publish must
// always be empty when subsequently acquired.  A stale buffered EntryID would
// corrupt the return value of the next publish that reuses the channel.
func TestRespChanAlwaysEmptyOnAcquire(t *testing.T) {
	q := NewQueue[int](logp.NewNopLogger(), nil, Settings{
		Events:        64,
		MaxGetRequest: 64,
		FlushTimeout:  time.Millisecond,
	}, 0, nil)
	defer q.Close(true)

	p := q.Producer(queue.ProducerConfig{})
	defer p.Close()

	// Exercise the full publish → broker-response → pool.Put lifecycle to
	// ensure at least one channel has completed a round-trip through the pool.
	const n = 20
	for range n {
		_, ok := p.Publish(0)
		require.True(t, ok)
	}
	batch, err := q.Get(n)
	require.NoError(t, err)
	batch.Done()

	// A channel acquired from the pool immediately after the above publishes
	// must have no buffered value.  If defer pool.Put were racing with
	// handlePendingResponse, the channel could carry a stale EntryID here.
	ch := getRespChan()
	require.NotNil(t, ch, "getRespChan must never return nil")
	select {
	case v := <-ch:
		t.Fatalf("pooled channel contained stale EntryID %v: "+
			"pool.Put must not fire before handlePendingResponse drains the channel", v)
	default:
		// correct — channel is empty
	}
	respChanPool.Put(ch)
}

// TestRespChanPoolNoAllocs verifies that getRespChan + pool.Put allocates
// nothing in steady state, i.e. channels are genuinely reused rather than
// freshly allocated on every publish.
//
// Without the pool: each publish calls make(chan queue.EntryID, 1) → 1 heap
// alloc per publish.
// With the pool (warm): the channel comes from the pool → 0 heap allocs.
func TestRespChanPoolNoAllocs(t *testing.T) {
	// Disable GC for the duration of the measurement so the pool cannot be
	// swept between iterations, making the result fully deterministic.
	oldGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGCPercent)

	// Prime the pool with one channel.
	ch := getRespChan()
	respChanPool.Put(ch)

	allocs := testing.AllocsPerRun(1000, func() {
		c := getRespChan()
		respChanPool.Put(c)
	})

	assert.Equal(t, 0.0, allocs,
		"getRespChan should return a pooled channel (0 allocs) after warmup; "+
			"got %.1f allocs/op — channels may not be returned to the pool", allocs)
}
