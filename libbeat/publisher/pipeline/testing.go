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
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/menderesk/beats/v7/libbeat/outputs"
	"github.com/menderesk/beats/v7/libbeat/publisher"
)

type mockPublishFn func(publisher.Batch) error

func newMockClient(publishFn mockPublishFn) outputs.Client {
	return &mockClient{publishFn: publishFn}
}

type mockClient struct {
	publishFn mockPublishFn
}

func (c *mockClient) String() string { return "mock_client" }
func (c *mockClient) Close() error   { return nil }
func (c *mockClient) Publish(_ context.Context, batch publisher.Batch) error {
	return c.publishFn(batch)
}

func newMockNetworkClient(publishFn mockPublishFn) outputs.Client {
	return &mockNetworkClient{newMockClient(publishFn)}
}

type mockNetworkClient struct {
	outputs.Client
}

func (c *mockNetworkClient) Connect() error { return nil }

type mockBatch struct {
	mu     sync.Mutex
	events []publisher.Event

	onEvents    func()
	onACK       func()
	onDrop      func()
	onRetry     func()
	onCancelled func()
}

func (b *mockBatch) Events() []publisher.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	signalFn(b.onEvents)
	return b.events
}

func (b *mockBatch) ACK()       { signalFn(b.onACK) }
func (b *mockBatch) Drop()      { signalFn(b.onDrop) }
func (b *mockBatch) Retry()     { signalFn(b.onRetry) }
func (b *mockBatch) Cancelled() { signalFn(b.onCancelled) }

func (b *mockBatch) RetryEvents(events []publisher.Event) {
	b.updateEvents(events)
	signalFn(b.onRetry)
}

func (b *mockBatch) updateEvents(events []publisher.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = events
}

func (b *mockBatch) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}

func (b *mockBatch) withRetryer(r standaloneRetryer) *mockBatch {
	wrapper := &mockBatch{
		events: b.events,
		onACK:  b.onACK,
		onDrop: b.onDrop,
	}
	wrapper.onRetry = func() { r.retryChan <- wrapper }
	wrapper.onCancelled = func() { r.retryChan <- wrapper }
	return wrapper
}

// standaloneRetryer is a helper that can be used to simulate retry
// behavior when unit testing outputWorker without a full pipeline. (In
// a live pipeline the retry calls are handled by the eventConsumer).
type standaloneRetryer struct {
	workQueue chan publisher.Batch
	retryChan chan publisher.Batch
	done      chan struct{}
}

func newStandaloneRetryer(workQueue chan publisher.Batch) standaloneRetryer {
	sr := standaloneRetryer{
		workQueue: workQueue,
		retryChan: make(chan publisher.Batch),
		done:      make(chan struct{}),
	}
	go sr.run()
	return sr
}

func (sr standaloneRetryer) run() {
	var batches []publisher.Batch
	for {
		var active publisher.Batch
		var outChan chan publisher.Batch
		// If we have a batch to send, set the batch and output channel.
		// Otherwise they'll be nil, and the select statement below will
		// ignore them.
		if len(batches) > 0 {
			active = batches[0]
			outChan = sr.workQueue
		}
		select {
		case batch := <-sr.retryChan:
			batches = append(batches, batch)
		case outChan <- active:
			batches = batches[1:]
		case <-sr.done:
			return
		}
	}
}

func (sr standaloneRetryer) close() {
	close(sr.done)
}

func signalFn(fn func()) {
	if fn != nil {
		fn()
	}
}

func randomBatch(min, max int) *mockBatch {
	return &mockBatch{
		events: make([]publisher.Event, randIntBetween(min, max)),
	}
}

// randIntBetween returns a random integer in [min, max)
func randIntBetween(min, max int) int {
	return rand.Intn(max-min) + min
}

func waitUntilTrue(duration time.Duration, fn func() bool) bool {
	end := time.Now().Add(duration)
	for time.Now().Before(end) {
		if fn() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
