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
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// The benchmarks below compare two production-realistic configurations:
//
//   - memqueue (normal Beat mode): a single shared memqueue with M producers
//     (one per input) all feeding the same FIFO, and one consumer worker
//     loop. This is what `filebeat` looks like today with M inputs.
//
//   - slabqueue (Beat-receiver mode): one slabqueue.Pool of the same total
//     capacity, but with M Queue façades — one per receiver — each with its
//     own producer and its own consumer goroutine. This is what the receiver
//     path looks like with M receivers.
//
// Both configurations process the same total number of events per iteration
// (benchEventsPerIteration), so ns/op values are directly comparable across
// rows: a row with N inputs/receivers does the same amount of work as every
// other row, just distributed differently. Both configurations use the same
// Settings.Events cap, so the operator-visible "max events in memory" budget
// is identical.
//
// Reported via -benchmem: ns/op (total iteration wall time), B/op, allocs/op.
//
// Run locally with:
//
//   go test -run=^$ -bench=. -benchmem -benchtime=2s ./libbeat/publisher/queue/slabqueue/...

type benchEvent struct {
	id int
}

// benchEventPayloadSize is the in-memory size of one benchEvent — used by
// b.SetBytes so go test -bench reports a MB/s column alongside ns/op.
// It tracks the size of the struct stored in pool.storage, not any
// production event payload; the number is honest for the benchmark but
// doesn't translate to "MB/s through the queue at production scale."
const benchEventPayloadSize = int64(unsafe.Sizeof(benchEvent{}))

// benchPipelines is the set of input/receiver counts we sweep. For memqueue
// this is the number of producers feeding the single shared queue. For
// slabqueue it is the number of Queue façades (one per receiver).
var benchPipelines = []int{1, 4, 8, 16}

// benchTotalCapacity is the queue/pool's total event budget — the same value
// for both configurations so the memory bound is identical.
const benchTotalCapacity = 4096

// benchEventsPerIteration is the constant total event count processed per
// b.N iteration. Whatever the input/receiver count, each iteration produces
// and consumes exactly this many events, so ns/op rows can be compared
// directly. Each producer publishes benchEventsPerIteration/M events.
const benchEventsPerIteration = 2048

// BenchmarkMemqueueShared models normal Beat mode: every input pushes into
// one shared memqueue, and a single consumer goroutine drains it. inputs=N
// means N producer goroutines feeding one queue.
func BenchmarkMemqueueShared(b *testing.B) {
	for _, m := range benchPipelines {
		b.Run(fmt.Sprintf("inputs=%d", m), func(b *testing.B) {
			settings := memqueue.Settings{
				Events:        benchTotalCapacity,
				MaxGetRequest: 128,
				FlushTimeout:  10 * time.Millisecond,
			}
			q := memqueue.NewQueue[benchEvent](
				logp.NewNopLogger(),
				queue.NewQueueObserver(nil),
				settings,
				0, nil,
			)

			producers := make([]queue.Producer[benchEvent], m)
			for i := range producers {
				producers[i] = q.Producer(queue.ProducerConfig{})
			}

			// One consumer drains the shared queue.
			runWorkload(b, producers, []queue.Queue[benchEvent]{q}, func() { _ = q.Close(true) })
		})
	}
}

// BenchmarkSlabQueuePool models multi-pipeline mode: one shared pool
// with the same total capacity, but each pipeline gets its own Queue
// façade and its own consumer goroutine. receivers=N means N façades,
// N producers, N consumers.
func BenchmarkSlabQueuePool(b *testing.B) {
	for _, m := range benchPipelines {
		b.Run(fmt.Sprintf("receivers=%d", m), func(b *testing.B) {
			pool := slabqueue.NewPool[benchEvent](
				slabqueue.Settings{Events: benchTotalCapacity}, nil,
			)

			queues := make([]queue.Queue[benchEvent], m)
			producers := make([]queue.Producer[benchEvent], m)
			for i := 0; i < m; i++ {
				queues[i] = pool.Connect()
				producers[i] = queues[i].Producer(queue.ProducerConfig{})
			}

			runWorkload(b, producers, queues, func() {
				for _, q := range queues {
					_ = q.Close(true)
				}
				pool.Shutdown()
			})
		})
	}
}

// runWorkload drives M producers and N consumers (where N = len(consumerQueues))
// over the configured pipeline count, then waits for every event to be drained.
// closeQueues is called after the timed workload completes; it must close every
// queue so the consumer goroutines unblock from their Get calls with io.EOF.
func runWorkload(b *testing.B, producers []queue.Producer[benchEvent], consumerQueues []queue.Queue[benchEvent], closeQueues func()) {
	m := len(producers)
	totalEvents := benchEventsPerIteration * b.N
	perProducer := totalEvents / m
	// Reports MB/s alongside ns/op so workload throughput is visible
	// in the benchmark output. Counts the in-memory size of benchEvent
	// (not a production payload size).
	b.SetBytes(int64(benchEventsPerIteration) * benchEventPayloadSize)

	consumed := make(chan int, 1024)

	var consumerWG sync.WaitGroup
	consumerWG.Add(len(consumerQueues))
	for _, q := range consumerQueues {
		q := q
		go func() {
			defer consumerWG.Done()
			for {
				batch, err := q.Get(128)
				if errors.Is(err, io.EOF) {
					return
				}
				if err != nil {
					b.Errorf("Get returned error: %v", err)
					return
				}
				n := batch.Count()
				batch.FreeEntries()
				batch.Done()
				consumed <- n
			}
		}()
	}

	b.ResetTimer()

	var producerWG sync.WaitGroup
	producerWG.Add(m)
	for i := 0; i < m; i++ {
		prod := producers[i]
		go func() {
			defer producerWG.Done()
			for j := 0; j < perProducer; j++ {
				prod.Publish(benchEvent{id: j})
			}
		}()
	}

	// Drain ack signals on the main goroutine. This avoids any extra
	// synchronization in the consumer hot path.
	got := 0
	for got < totalEvents {
		got += <-consumed
	}

	b.StopTimer()

	producerWG.Wait()
	// Close the queues so the consumer goroutines unblock from Get with io.EOF.
	closeQueues()
	consumerWG.Wait()
}

// BenchmarkProducerThroughput mirrors memqueue's BenchmarkProducerThroughput
// (libbeat/publisher/queue/memqueue/queue_test.go) for direct EPS
// comparison: 10 producer goroutines feed a single 10,000-slot queue
// while one consumer drains batches.
func BenchmarkProducerThroughput(b *testing.B) {
	const queueSize = 10000
	const publishWorkers = 10

	pool := slabqueue.NewPool[int](slabqueue.Settings{Events: queueSize}, nil)
	defer pool.Shutdown()
	testQueue := pool.Connect()

	ctx, cancel := context.WithCancel(context.Background())
	publishWorker := func() {
		producer := testQueue.Producer(queue.ProducerConfig{})
		for ctx.Err() == nil {
			producer.Publish(0)
		}
	}
	for range publishWorkers {
		go publishWorker()
	}
	var totalEvents int64
	for b.Loop() {
		batch, err := testQueue.Get(queueSize)
		if err != nil {
			b.Fatal("Fetching queue batch should succeed")
		}
		totalEvents += int64(batch.Count())
		batch.Done()
	}
	if elapsed := b.Elapsed().Seconds(); elapsed > 0 {
		b.ReportMetric(float64(totalEvents)/elapsed, "events/s")
	}
	cancel()
	_ = testQueue.Close(true)
}
