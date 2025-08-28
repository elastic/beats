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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/publisher"

	"go.elastic.co/apm/v2"

	"github.com/elastic/beats/v7/libbeat/outputs"
)

type worker struct {
	qu     chan publisher.Batch
	cancel func()
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

	logger logger

	tracer *apm.Tracer
}

func makeClientWorker(qu chan publisher.Batch, client outputs.Client, logger logger, tracer *apm.Tracer) outputWorker {
	ctx, cancel := context.WithCancel(context.Background())
	w := worker{
		qu:     qu,
		cancel: cancel,
	}

	var c interface {
		outputWorker
		run(context.Context)
	}

	if nc, ok := client.(outputs.NetworkClient); ok {
		c = &netClientWorker{
			worker: w,
			client: nc,
			logger: logger,
			tracer: tracer,
		}
	} else {
		c = &clientWorker{worker: w, client: client}
	}

	go c.run(ctx)
	return c
}

func (w *worker) close() {
	w.cancel()
}

func (w *clientWorker) Close() error {
	w.worker.close()
	return w.client.Close()
}

func (w *clientWorker) run(ctx context.Context) {
	for {
		// We wait for either the worker to be closed or for there to be a batch of
		// events to publish.
		select {

		case <-ctx.Done():
			return

		case batch := <-w.qu:
			if batch == nil {
				continue
			}
			if err := w.client.Publish(ctx, batch); err != nil {
				return
			}
		}
	}
}

func (w *netClientWorker) Close() error {
	w.worker.close()
	return w.client.Close()
}

func (w *netClientWorker) run(ctx context.Context) {
	var (
		connected         = false
		reconnectAttempts = 0
	)

	for {
		// We wait for either the worker to be closed or for there to be a batch of
		// events to publish.
		select {

		case <-ctx.Done():
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

				err := w.client.Connect(ctx)
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

			if err := w.publishBatch(ctx, batch); err != nil {
				connected = false
			}
		}
	}
}

func (w *netClientWorker) publishBatch(ctx context.Context, batch publisher.Batch) error {
	if w.tracer != nil && w.tracer.Recording() {
		tx := w.tracer.StartTransaction("publish", "output")
		defer tx.End()
		tx.Context.SetLabel("worker", "netclient")
		ctx = apm.ContextWithTransaction(ctx, tx)
	}
	err := w.client.Publish(ctx, batch)
	if err != nil {
		err = fmt.Errorf("failed to publish events: %w", err)
		apm.CaptureError(ctx, err).Send()
		w.logger.Error(err)
		// on error return to connect loop
		return err
	}
	return nil
}
