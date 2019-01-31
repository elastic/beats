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
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

// clientWorker manages output client of type outputs.Client, not supporting reconnect.
type clientWorker struct {
	observer outputObserver
	qu       workQueue
	client   outputs.Client
	closed   atomic.Bool
}

// netClientWorker manages reconnectable output clients of type outputs.NetworkClient.
type netClientWorker struct {
	observer outputObserver
	qu       workQueue
	client   outputs.NetworkClient
	done     chan struct{}

	batchSize  int
	batchSizer func() int
}

func makeClientWorker(observer outputObserver, qu workQueue, client outputs.Client) outputWorker {
	if nc, ok := client.(outputs.NetworkClient); ok {
		c := &netClientWorker{observer: observer, qu: qu, client: nc, done: make(chan struct{})}
		go c.run()
		return c
	}
	c := &clientWorker{observer: observer, qu: qu, client: client}
	go c.run()
	return c
}

func (w *clientWorker) Close() error {
	w.closed.Store(true)
	return w.client.Close()
}

func (w *clientWorker) run() {
	for !w.closed.Load() {
		for batch := range w.qu {
			w.observer.outBatchSend(len(batch.events))
			if err := w.client.Publish(batch); err != nil {
				return
			}
		}
	}
}

func (w *netClientWorker) Close() error {
	close(w.done)
	return w.client.Close()
}

func (w *netClientWorker) run() {
	var (
		connected         bool
		reconnectAttempts int
	)

	for {
		select {
		case <-w.done:
			logp.Info("Closed connection to %v", w.client)
			return
		case batch := <-w.qu:
			if batch == nil {
				continue
			}

			if !connected {
				batch.Cancelled()

				if reconnectAttempts > 0 {
					logp.Info("Attempting to reconnect to %v with %d reconnect attempt(s)", w.client, reconnectAttempts)
				} else {
					logp.Info("Connecting to %v", w.client)
				}

				if err := w.client.Connect(); err != nil {
					logp.Err("Failed to connect to %v: %v", w.client, err)
					reconnectAttempts++
				}

				logp.Info("Connection to %v established", w.client)
				connected = true
				reconnectAttempts = 0
				continue
			}

			err := w.client.Publish(batch)
			if err != nil {
				logp.Err("Failed to publish events: %v", err)
				// on error return to connect loop
				connected = false
			}
		}
	}
}
