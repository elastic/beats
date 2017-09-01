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
	closed   atomic.Bool

	batchSize  int
	batchSizer func() int
}

func makeClientWorker(observer outputObserver, qu workQueue, client outputs.Client) outputWorker {
	if nc, ok := client.(outputs.NetworkClient); ok {
		c := &netClientWorker{observer: observer, qu: qu, client: nc}
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
	w.closed.Store(true)
	return w.client.Close()
}

func (w *netClientWorker) run() {
	for !w.closed.Load() {
		// start initial connect loop from first batch, but return
		// batch to pipeline for other outputs to catch up while we're trying to connect
		for batch := range w.qu {
			batch.Cancelled()

			if w.closed.Load() {
				return
			}

			err := w.client.Connect()
			if err != nil {
				logp.Err("Failed to connect: %v", err)
				continue
			}

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

			err := w.client.Publish(batch)
			if err != nil {
				logp.Err("Failed to publish events: %v", err)
				// on error return to connect loop
				break
			}
		}
	}
}
