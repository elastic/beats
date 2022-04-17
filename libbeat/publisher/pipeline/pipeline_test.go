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

	"github.com/menderesk/beats/v7/libbeat/common/atomic"
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue"
)

type testQueue struct {
	close        func() error
	bufferConfig func() queue.BufferConfig
	producer     func(queue.ProducerConfig) queue.Producer
	consumer     func() queue.Consumer
}

type testProducer struct {
	publish func(try bool, event publisher.Event) bool
	cancel  func() int
}

type testConsumer struct {
	get   func(sz int) (queue.Batch, error)
	close func() error
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

func (q *testQueue) Consumer() queue.Consumer {
	if q.consumer != nil {
		return q.consumer()
	}
	return nil
}

func (p *testProducer) Publish(event publisher.Event) bool {
	if p.publish != nil {
		return p.publish(false, event)
	}
	return false
}

func (p *testProducer) TryPublish(event publisher.Event) bool {
	if p.publish != nil {
		return p.publish(true, event)
	}
	return false
}

func (p *testProducer) Cancel() int {
	if p.cancel != nil {
		return p.cancel()
	}
	return 0
}

func (p *testConsumer) Get(sz int) (queue.Batch, error) {
	if p.get != nil {
		return p.get(sz)
	}
	return nil, nil
}

func (p *testConsumer) Close() error {
	if p.close() != nil {
		return p.close()
	}
	return nil
}

func makeBlockingQueue() queue.Queue {
	return makeTestQueue(emptyConsumer, blockingProducer)
}

func makeTestQueue(
	makeConsumer func() queue.Consumer,
	makeProducer func(queue.ProducerConfig) queue.Producer,
) queue.Queue {
	var mux sync.Mutex
	var wg sync.WaitGroup
	consumers := map[*testConsumer]struct{}{}
	producers := map[queue.Producer]struct{}{}

	return &testQueue{
		close: func() error {
			mux.Lock()
			for consumer := range consumers {
				consumer.Close()
			}
			for producer := range producers {
				producer.Cancel()
			}
			mux.Unlock()

			wg.Wait()
			return nil
		},

		consumer: func() queue.Consumer {
			var consumer *testConsumer
			c := makeConsumer()
			consumer = &testConsumer{
				get: func(sz int) (queue.Batch, error) { return c.Get(sz) },
				close: func() error {
					err := c.Close()

					mux.Lock()
					defer mux.Unlock()
					delete(consumers, consumer)
					wg.Done()

					return err
				},
			}

			mux.Lock()
			defer mux.Unlock()
			consumers[consumer] = struct{}{}
			wg.Add(1)
			return consumer
		},

		producer: func(cfg queue.ProducerConfig) queue.Producer {
			var producer *testProducer
			p := makeProducer(cfg)
			producer = &testProducer{
				publish: func(try bool, event publisher.Event) bool {
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

func emptyConsumer() queue.Consumer {
	done := make(chan struct{})
	return &testConsumer{
		get: func(sz int) (queue.Batch, error) {
			<-done
			return nil, nil
		},
		close: func() error {
			close(done)
			return nil
		},
	}
}

func blockingProducer(_ queue.ProducerConfig) queue.Producer {
	sig := make(chan struct{})
	waiting := atomic.MakeInt(0)

	return &testProducer{
		publish: func(_ bool, _ publisher.Event) bool {
			waiting.Inc()
			<-sig
			return false
		},

		cancel: func() int {
			close(sig)
			return waiting.Load()
		},
	}
}
