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
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
)

var _workerID atomic.Uint

func lf(msg string, v ...interface{}) {
	now := time.Now().Format("15:04:05.00000")
	fmt.Printf(now+" "+msg+"\n", v...)
}

func (w *worker) lf(msg string, v ...interface{}) {
	lf("[worker "+strconv.Itoa(int(w.id))+"] "+msg, v...)
}

type worker struct {
	id       uint
	observer outputObserver
	qu       workQueue
	done     chan struct{}
	inFlight chan struct{}
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
		id:       _workerID.Inc(),
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

	//w.lf("starting...")
	go c.run()
	return c
}

func (w *worker) close() {
	close(w.done)
	//lf("w.inFlight == nil: %#v", w.inFlight == nil)
	if w.inFlight != nil {
		//lf("waiting for inflight events to publish")
		<-w.inFlight
		//lf("inflight events published")
	}
	//w.lf("closed")
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
			w.observer.outBatchSend(len(batch.events))

			w.inFlight = make(chan struct{})
			if err := w.client.Publish(batch); err != nil {
				close(w.inFlight)
				return
			}
			close(w.inFlight)
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
			//lf("got done signal")
			return

		case batch, ok := <-w.qu:
			if !ok {
				//w.lf("workqueue closed")
			}
			if batch == nil {
				continue
			}

			// Try to (re)connect so we can publish batch
			if !connected {
				// Return batch to other output workers while we try to (re)connect
				//w.lf("canceling batch of %v events", len(batch.Events()))
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

			//w.lf("about to publish %v events", len(batch.Events()))
			w.inFlight = make(chan struct{})
			if err := w.client.Publish(batch); err != nil {
				close(w.inFlight)
				w.logger.Errorf("Failed to publish events: %v", err)
				// on error return to connect loop
				connected = false
			}
			close(w.inFlight)
		}
	}
}
