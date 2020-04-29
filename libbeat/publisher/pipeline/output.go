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
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

type worker struct {
	id       uint
	observer outputObserver
	qu       workQueue
	done     chan struct{}
}

// clientWorker manages output client of type outputs.Client, not supporting reconnect.
type clientWorker struct {
	worker
	client outputs.Client
}

// netClientWorker manages reconnectable output clients of type outputs.NetworkClient.
type netClientWorker struct {
	worker
	client outputs.NetworkClient

	batchSize  int
	batchSizer func() int
	logger     *logp.Logger
}

func makeClientWorker(observer outputObserver, qu workQueue, client outputs.Client) outputWorker {
	w := worker{
		observer: observer,
		qu:       qu,
		done:     make(chan struct{}),
	}

	var c interface {
		outputWorker
		run()
	}

	if nc, ok := client.(outputs.NetworkClient); ok {
		c = &netClientWorker{
			worker: w,
			client: nc,
			logger: logp.NewLogger("publisher_pipeline_output"),
		}
	} else {
		c = &clientWorker{worker: w, client: client}
	}

	go c.run()
	return c
}

func (w *worker) close() {
	close(w.done)
}

func (w *clientWorker) Close() error {
	w.worker.close()
	return w.client.Close()
}

func (w *clientWorker) run() {
	for {
		// We wait for either the worker to be closed or for there to be a batch of
		// events to publish.
		select {

		case <-w.done:
			return

		case batch := <-w.qu:
			if batch == nil {
				continue
			}
			w.observer.outBatchSend(len(batch.Events()))

			if err := w.client.Publish(batch); err != nil {
				return
			}
		}
	}
}

func (w *netClientWorker) Close() error {
	w.worker.close()
	return w.client.Close()
}

func (w *netClientWorker) run() {
	var (
		connected         = false
		reconnectAttempts = 0
	)

	for {
		// We wait for either the worker to be closed or for there to be a batch of
		// events to publish.
		select {

		case <-w.done:
			return

		case batch := <-w.qu:
			if batch == nil {
				continue
			}

			// Try to (re)connect so we can publish batch
			if !connected {
				// Return batch to other output workers while we try to (re)connect
				batch.Cancelled()

				if reconnectAttempts == 0 {
					w.logger.Infof("Connecting to %v", w.client)
				} else {
					w.logger.Infof("Attempting to reconnect to %v with %d reconnect attempt(s)", w.client, reconnectAttempts)
				}

				err := w.client.Connect()
				connected = err == nil
				if connected {
					w.logger.Infof("Connection to %v established", w.client)
					reconnectAttempts = 0
				} else {
					w.logger.Errorf("Failed to connect to %v: %v", w.client, err)
					reconnectAttempts++
				}

				continue
			}

			if err := w.client.Publish(batch); err != nil {
				w.logger.Errorf("Failed to publish events: %v", err)
				// on error return to connect loop
				connected = false
			}
		}
	}
}
