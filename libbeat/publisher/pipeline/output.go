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
	"errors"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

// clientWorker manages output client of type outputs.Client, not supporting reconnect.
type clientWorker struct {
	observer outputObserver
	qu       workQueue
	client   outputs.Client
	done     chan struct{}
	wg       sync.WaitGroup
}

// netClientWorker manages reconnectable output clients of type outputs.NetworkClient.
type netClientWorker struct {
	observer outputObserver
	qu       workQueue
	client   netClient
	done     chan struct{}
	wg       sync.WaitGroup

	batchSize  int
	batchSizer func() int
}

type netClient struct {
	client outputs.NetworkClient

	mu     sync.Mutex
	err    error
	active bool
}

var errOutputDisabled = errors.New("output disabled")

func makeClientWorker(observer outputObserver, qu workQueue, client outputs.Client) outputWorker {
	if nc, ok := client.(outputs.NetworkClient); ok {
		c := &netClientWorker{observer: observer, qu: qu, client: makeNetClient(nc)}
		c.start()
		return c
	}
	c := &clientWorker{observer: observer, qu: qu, client: client}
	go c.run()
	return c
}

func (w *clientWorker) Close() error {
	close(w.done)
	err := w.client.Close()
	w.wg.Wait()
	return err
}

func (w *clientWorker) start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run()
	}()
}

func (w *clientWorker) run() {
	for w.active() {
		batch, ok := w.next()
		if !ok {
			return
		}

		w.observer.outBatchSend(len(batch.events))
		if err := w.client.Publish(batch); err != nil {
			return
		}
	}
}

func (w *clientWorker) active() bool {
	select {
	case <-w.done:
		return false
	default:
		return true
	}
}

func (w *clientWorker) next() (*Batch, bool) {
	for {
		select {
		case <-w.done:
			return nil, false
		case b := <-w.qu:
			if b != nil {
				return b, true
			}
		}
	}
}

func (w *netClientWorker) Close() error {
	close(w.done)
	err := w.client.Disable() // async close and disable client from reconnecting
	w.wg.Wait()
	return err
}

func (w *netClientWorker) start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run()
	}()
}

func (w *netClientWorker) run() {
	var (
		connected         bool
		reconnectAttempts int
	)

	for w.active() {
		batch, ok := w.next()
		if !ok {
			break
		}

		if !connected {
			batch.Cancelled()
			if reconnectAttempts > 0 {
				logp.Info("Attempting to reconnect to %v with %d reconnect attempt(s)", w.client.String(), reconnectAttempts)
			} else {
				logp.Info("Connecting to %v", w.client.String())
			}

			err := w.connect()
			connected = err == nil
			if connected {
				reconnectAttempts = 0
			} else {
				reconnectAttempts++
			}
			continue
		}

		err := w.client.Publish(batch)
		if err != nil {
			logp.Err("Failed to publish events: %v", err)
			// on error return to connect loop
			connected = false

			w.client.Close()
		}
	}
}

func (w *netClientWorker) next() (*Batch, bool) {
	for {
		select {
		case <-w.done:
			return nil, false
		case b := <-w.qu:
			if b != nil {
				return b, true
			}
		}
	}
}

func (w *netClientWorker) connect() error {
	err := w.client.Connect()
	if err != nil {
		logp.Err("Failed to connect to %v: %v", w.client.String(), err)
	} else {
		logp.Info("Connection to %v established", w.client.String())
	}
	return err
}

func (w *netClientWorker) active() bool {
	select {
	case <-w.done:
		return false
	default:
		return true
	}
}

func makeNetClient(c outputs.NetworkClient) netClient {
	return netClient{
		client: c,
		active: true,
	}
}

func (c *netClient) Disable() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return c.err
	}

	c.active = false
	err := c.Close()
	if err != nil {
		c.err = err
	}
	return err
}

func (c *netClient) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.err
}

func (c *netClient) String() string {
	return c.client.String()
}

func (c *netClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return c.err
	}
	return c.client.Close()
}

func (c *netClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.active {
		return errOutputDisabled
	}

	c.mu.Unlock()
	err := c.Connect()
	c.mu.Lock()

	if !c.active {
		if err == nil {
			// connection has been closed concurrently during Connect
			// attempt to close in case of race
			c.updErr(c.Close())
		}
		err = errOutputDisabled
	}
	return err
}

func (c *netClient) updErr(err error) {
	if c.err == nil && err != nil {
		c.err = err
	}
}

func (c *netClient) Publish(batch *Batch) error {
	if !c.isActive() {
		return errOutputDisabled
	}

	// assume that concurrent Close or close before Publish call becomes
	// effective making Publish fail immediately.
	return c.client.Publish(batch)
}

func (c *netClient) isActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}
