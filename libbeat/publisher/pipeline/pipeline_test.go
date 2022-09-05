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

package pipeline

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type testQueue struct {
	close        func() error
	bufferConfig func() queue.BufferConfig
	producer     func(queue.ProducerConfig) queue.Producer
	get          func(sz int) (queue.Batch, error)
}

type testProducer struct {
	publish func(try bool, event interface{}) (queue.EntryID, bool)
	cancel  func() int
}

func (q *testQueue) Metrics() (queue.Metrics, error) {
	return queue.Metrics{}, nil
}

func (q *testQueue) Close() error {
	if q.close != nil {
		return q.close()
	}
	return nil
}

func (q *testQueue) BufferConfig() queue.BufferConfig {
	if q.bufferConfig != nil {
		return q.bufferConfig()
	}
	return queue.BufferConfig{}
}

func (q *testQueue) Producer(cfg queue.ProducerConfig) queue.Producer {
	if q.producer != nil {
		return q.producer(cfg)
	}
	return nil
}

func (q *testQueue) Get(sz int) (queue.Batch, error) {
	if q.get != nil {
		return q.get(sz)
	}
	return nil, nil
}

func (p *testProducer) Publish(event interface{}) (queue.EntryID, bool) {
	if p.publish != nil {
		return p.publish(false, event)
	}
	return 0, false
}

func (p *testProducer) TryPublish(event interface{}) (queue.EntryID, bool) {
	if p.publish != nil {
		return p.publish(true, event)
	}
	return 0, false
}

func (p *testProducer) Cancel() int {
	if p.cancel != nil {
		return p.cancel()
	}
	return 0
}

func makeTestQueue() queue.Queue {
	var mux sync.Mutex
	var wg sync.WaitGroup
	producers := map[queue.Producer]struct{}{}

	return &testQueue{
		close: func() error {
			mux.Lock()
			for producer := range producers {
				producer.Cancel()
			}
			mux.Unlock()

			wg.Wait()
			return nil
		},
		get: func(count int) (queue.Batch, error) {
			//<-done
			return nil, nil
		},

		producer: func(cfg queue.ProducerConfig) queue.Producer {
			var producer *testProducer
			p := blockingProducer(cfg)
			producer = &testProducer{
				publish: func(try bool, event interface{}) (queue.EntryID, bool) {
					if try {
						return p.TryPublish(event)
					}
					return p.Publish(event)
				},
				cancel: func() int {
					i := p.Cancel()

					mux.Lock()
					defer mux.Unlock()
					delete(producers, producer)
					wg.Done()

					return i
				},
			}

			mux.Lock()
			defer mux.Unlock()
			producers[producer] = struct{}{}
			wg.Add(1)
			return producer
		},
	}
}

func blockingProducer(_ queue.ProducerConfig) queue.Producer {
	sig := make(chan struct{})
	waiting := atomic.MakeInt(0)

	return &testProducer{
		publish: func(_ bool, _ interface{}) (queue.EntryID, bool) {
			waiting.Inc()
			<-sig
			return 0, false
		},

		cancel: func() int {
			close(sig)
			return waiting.Load()
		},
	}
}
