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

package proxyqueue

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
)

func TestQueueStuff(t *testing.T) {
	// THIS PART IS NOT DONE YET (but it does correctly test some basic things)
	var acked atomic.Int
	logger := logp.NewLogger("proxy-queue-tests")
	// Create a proxy queue where each batch is at most 2 events
	testQueue := NewQueue(logger, Settings{BatchSize: 2})
	defer testQueue.Close()

	producer := testQueue.Producer(queue.ProducerConfig{
		ACK: func(count int) {
			acked.Add(count)
			fmt.Printf("got ack %d\n", count)
		},
	})
	// Try to publish 3 events, only the first two should succeed until we read a batch
	_, success := producer.TryPublish(1)
	assert.True(t, success)
	_, success = producer.TryPublish(2)
	assert.True(t, success)
	_, success = producer.TryPublish(3)
	assert.False(t, success, "Current batch should only fit two events")

	assert.Equal(t, 0, acked.Load(), "No batches have been acked yet")
	batch, err := testQueue.Get(5)
	assert.NoError(t, err, "Should be able to read a batch")
	batch.Done()
	assert.Equal(t, 2, acked.Load(), "No batches have been acked yet")

}

/// limitations of proxy queue (to go in README):
// - doesn't use real queue.EntryID
// - doesn't implement producer cancel
// - doesn't respect requested (consumer) batch size, only the configured size
