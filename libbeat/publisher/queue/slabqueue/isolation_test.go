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

package slabqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// A receiver retains its slots until output acknowledges its events, including
// after Get removes them from its FIFO. A stalled output must not consume the
// configured capacity of another receiver whose queue is empty.
func TestCappedQueueDoesNotStarveEmptyReceiver(t *testing.T) {
	pool := NewPool[int](Settings{Events: 4}, nil)
	t.Cleanup(pool.Shutdown)
	stalled := pool.Connect()
	stalled.SetTarget(4)
	healthy := pool.Connect()
	healthy.SetTarget(2)
	p1 := stalled.Producer(queue.ProducerConfig{})
	p2 := healthy.Producer(queue.ProducerConfig{})

	for i := range 4 {
		_, ok := p1.TryPublish(i)
		require.True(t, ok, "stalled receiver should admit its configured event budget")
	}
	b, err := stalled.Get(0)
	require.NoError(t, err, "stalled receiver should hand its events to the output")
	t.Cleanup(b.Done)
	require.Equal(t, 4, b.Count(), "all stalled events should remain in flight")

	for i := range 2 {
		_, ok := p2.TryPublish(100 + i)
		assert.True(t, ok, "an empty receiver must retain its full configured capacity while its sibling is stalled")
	}
}
