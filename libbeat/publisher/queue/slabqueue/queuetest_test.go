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

package slabqueue_test

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
)

func makeTestQueue(sz int) queuetest.QueueFactory {
	return func(t *testing.T) queue.Queue[publisher.Event] {
		pool := slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: sz}, nil)
		t.Cleanup(func() { pool.Shutdown() })
		return pool.Connect()
	}
}

func TestSlabQueueConformance(t *testing.T) {
	events := 4096
	batchSize := 100
	bufferSize := 8192

	// slabqueue's Queue façade is a single-consumer design: only one
	// goroutine may call Get concurrently. TestMultiProducerConsumer
	// includes cases with multiple concurrent consumers on the same
	// queue, which violates that contract. Each pipeline gets its own
	// façade in production, so concurrent consumers are not a supported
	// use case.
	t.Run("slabqueue", func(t *testing.T) {
		t.Parallel()
		queuetest.TestSingleProducerConsumer(t, events, batchSize, makeTestQueue(bufferSize))
	})
}
