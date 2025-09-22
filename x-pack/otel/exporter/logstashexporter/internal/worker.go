// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Work struct {
	batch  publisher.Batch
	result chan error
}

func NewWork(batch publisher.Batch) *Work {
	return &Work{
		batch:  batch,
		result: make(chan error, 1),
	}
}

func (w *Work) Result() chan error {
	return w.result
}

type Worker interface {
	Close() error
}

type worker struct {
	workQueue chan *Work
	cancel    func()
}

type clientWorker struct {
	worker
	client outputs.Client
}

type netClientWorker struct {
	worker
	client outputs.NetworkClient
	logger logp.Logger
}

func MakeClientWorker(workQueue chan *Work, client outputs.Client, logger logp.Logger) Worker {
	ctx, cancel := context.WithCancel(context.Background())
	w := worker{
		workQueue: workQueue,
		cancel:    cancel,
	}

	var c interface {
		Worker
		run(context.Context)
	}

	if nc, ok := client.(outputs.NetworkClient); ok {
		c = &netClientWorker{worker: w, client: nc, logger: logger}
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
	w.close()
	return w.client.Close()
}

func (w *clientWorker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case work := <-w.workQueue:
			work.result <- w.client.Publish(ctx, work.batch)
		}
	}
}

func (w *netClientWorker) Close() error {
	w.close()
	return w.client.Close()
}

func (w *netClientWorker) run(ctx context.Context) {
	var (
		connected         = false
		reconnectAttempts = 0
	)

	for {
		select {
		case <-ctx.Done():
			return
		case work := <-w.workQueue:
			if !connected {
				// Return the batch to other workers while it tries to reconnect
				work.batch.Cancelled()
				work.result <- nil

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
					w.logger.Errorf("Failed to connect to %v: %q", w.client, err)
					reconnectAttempts++
				}

				continue
			}

			if err := w.publishBatch(ctx, work.batch); err != nil {
				work.result <- err
				connected = false
			} else {
				work.result <- nil
			}
		}
	}
}

func (w *netClientWorker) publishBatch(ctx context.Context, batch publisher.Batch) error {
	err := w.client.Publish(context.WithoutCancel(ctx), batch)
	if err != nil {
		err = fmt.Errorf("failed to publish events: %w", err)
	}
	return err
}
