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
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

var _workerID atomic.Uint

func (w *worker) lf(msg string, v ...interface{}) {
	lf("[worker "+strconv.Itoa(int(w.id))+"] "+msg, v...)
}

func (w *worker) ln(msg string) {
	w.lf(msg + "\n")
}

type worker struct {
	id       uint
	observer outputObserver
	qu       workQueue
	closed   atomic.Bool
	inflight chan struct{}
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
	worker := worker{
		observer: observer,
		qu:       qu,
	}

	worker.id = _workerID.Inc()

	if nc, ok := client.(outputs.NetworkClient); ok {
		c := &netClientWorker{
			worker: worker,
			client: nc,
			logger: logp.NewLogger("publisher_pipeline_output"),
		}
		go c.run()
		return c
	}
	c := &clientWorker{
		worker: worker,
		client: client,
	}

	worker.ln("running client worker...")
	go c.run()
	return c
}

func (w *worker) close() {
	w.ln("worker asked to close")
	w.closed.Store(true)
	w.lf("w.inflight == nil: %#v\n", w.inflight == nil)
	if w.inflight != nil {
		w.ln("waiting for inflight events to publish")
		<-w.inflight
		w.ln("inflight events published")
	}
}

func (w *clientWorker) Close() error {
	w.close()
	return w.client.Close()
}

func (w *clientWorker) run() {
	defer func() {
		w.ln("clientWorker closed")
	}()
	for !w.closed.Load() {
		for batch := range w.qu {
			if w.closed.Load() {
				if batch != nil {
					w.lf("canceling batch of %v events\n", len(batch.events))
					batch.Cancelled()
				}
				return
			}

			if batch == nil {
				continue
			}

			w.observer.outBatchSend(len(batch.events))

			w.lf("in clientWorker, about to publish %v events\n", len(batch.events))
			w.inflight = make(chan struct{})
			if err := w.client.Publish(batch); err != nil {
				close(w.inflight)
				break
			}
			close(w.inflight)
		}
	}
}

func (w *netClientWorker) Close() error {
	w.close()
	return w.client.Close()
}

func (w *netClientWorker) run() {
	for !w.closed.Load() {
		reconnectAttempts := 0

		// start initial connect loop from first batch, but return
		// batch to pipeline for other outputs to catch up while we're trying to connect
		for batch := range w.qu {
			batch.Cancelled()

			if w.closed.Load() {
				w.logger.Infof("Closed connection to %v", w.client)
				return
			}

			if reconnectAttempts > 0 {
				w.logger.Infof("Attempting to reconnect to %v with %d reconnect attempt(s)", w.client, reconnectAttempts)
			} else {
				w.logger.Infof("Connecting to %v", w.client)
			}

			err := w.client.Connect()
			if err != nil {
				w.logger.Errorf("Failed to connect to %v: %v", w.client, err)
				reconnectAttempts++
				continue
			}

			w.logger.Infof("Connection to %v established", w.client)
			reconnectAttempts = 0
			break
		}

		// send loop
		for batch := range w.qu {
			if w.closed.Load() {
				if batch != nil {
					batch.Cancelled()
				}
				return
			}

			w.inflight = make(chan struct{})
			err := w.client.Publish(batch)
			if err != nil {
				close(w.inflight)
				w.logger.Errorf("Failed to publish events: %v", err)
				// on error return to connect loop
				break
			}
			close(w.inflight)
		}
	}
}
