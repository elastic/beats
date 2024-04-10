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

package logstash

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport"
	v2 "github.com/elastic/go-lumber/client/v2"
)

type asyncClient struct {
	log *logp.Logger
	*transport.Client
	observer outputs.Observer
	client   *v2.AsyncClient
	win      *window

	connect func() error

	mutex sync.Mutex
}

type msgRef struct {
	client    *asyncClient
	count     atomic.Uint32
	batch     publisher.Batch
	slice     []publisher.Event
	err       error
	win       *window
	batchSize int
}

func newAsyncClient(
	beat beat.Info,
	conn *transport.Client,
	observer outputs.Observer,
	config *Config,
) (*asyncClient, error) {

	log := logp.NewLogger("logstash")
	c := &asyncClient{
		log:      log,
		Client:   conn,
		observer: observer,
	}

	if config.SlowStart {
		c.win = newWindower(defaultStartMaxWindowSize, config.BulkMaxSize)
	}

	if config.TTL != 0 {
		log.Warn(`The async Logstash client does not support the "ttl" option`)
	}

	queueSize := config.Pipelining - 1
	timeout := config.Timeout
	compressLvl := config.CompressionLevel
	clientFactory := makeClientFactory(queueSize, timeout, logstashEventUnwrapper, compressLvl)

	var err error
	c.client, err = clientFactory(c.Client)
	if err != nil {
		return nil, err
	}

	c.connect = func() error {
		err := c.Client.Connect()
		if err == nil {
			c.client, err = clientFactory(c.Client)
		}
		return err
	}

	return c, nil
}

func makeClientFactory(
	queueSize int,
	timeout time.Duration,
	enc func(interface{}) ([]byte, error),
	compressLvl int,
) func(net.Conn) (*v2.AsyncClient, error) {
	return func(conn net.Conn) (*v2.AsyncClient, error) {
		return v2.NewAsyncClientWithConn(conn, queueSize,
			v2.JSONEncoder(enc),
			v2.Timeout(timeout),
			v2.CompressionLevel(compressLvl),
		)
	}
}

func (c *asyncClient) Connect() error {
	c.log.Debug("connect")
	return c.connect()
}

func (c *asyncClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.log.Debug("close connection")

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return c.Client.Close()
}

func (c *asyncClient) Publish(_ context.Context, batch publisher.Batch) error {
	st := c.observer
	events := batch.Events()
	st.NewBatch(len(events))

	if len(events) == 0 {
		batch.ACK()
		return nil
	}

	ref := &msgRef{
		client:    c,
		count:     atomic.MakeUint32(1),
		batch:     batch,
		slice:     events,
		batchSize: len(events),
		win:       c.win,
		err:       nil,
	}
	defer ref.dec()

	for len(events) > 0 {
		var (
			n   int
			err error
		)

		if c.win == nil {
			n = len(events)
			err = c.sendEvents(ref, events)
		} else {
			n, err = c.publishWindowed(ref, events)
		}

		c.log.Debugf("%v events out of %v events sent to logstash host %s. Continue sending",
			n, len(events), c.Host())

		events = events[n:]
		if err != nil {
			_ = c.Close()
			return err
		}
	}

	return nil
}

func (c *asyncClient) String() string {
	return "async(" + c.Client.String() + ")"
}

func (c *asyncClient) publishWindowed(
	ref *msgRef,
	events []publisher.Event,
) (int, error) {
	batchSize := len(events)
	windowSize := c.win.get()

	c.log.Debugf("Try to publish %v events to logstash host %s with window size %v",
		batchSize, c.Host(), windowSize)

	// prepare message payload
	if batchSize > windowSize {
		events = events[:windowSize]
	}

	err := c.sendEvents(ref, events)
	if err != nil {
		return 0, err
	}

	return len(events), nil
}

func (c *asyncClient) sendEvents(ref *msgRef, events []publisher.Event) error {
	client := c.getClient()
	if client == nil {
		return errors.New("connection closed")
	}
	window := make([]interface{}, len(events))
	for i := range events {
		window[i] = events[i].EncodedEvent
	}
	ref.count.Inc()
	return client.Send(ref.callback, window)
}

func (c *asyncClient) getClient() *v2.AsyncClient {
	c.mutex.Lock()
	client := c.client
	c.mutex.Unlock()
	return client
}

func (r *msgRef) callback(seq uint32, err error) {
	if err != nil {
		r.fail(seq, err)
	} else {
		r.done(seq)
	}
}

func (r *msgRef) done(n uint32) {
	r.client.observer.Acked(int(n))
	r.slice = r.slice[n:]
	if r.win != nil {
		r.win.tryGrowWindow(r.batchSize)
	}
	r.dec()
}

func (r *msgRef) fail(n uint32, err error) {
	if r.err == nil {
		r.err = err
	}
	r.slice = r.slice[n:]
	if r.win != nil {
		r.win.shrinkWindow()
	}

	r.client.observer.Acked(int(n))

	r.dec()
}

func (r *msgRef) dec() {
	i := r.count.Dec()
	if i > 0 {
		return
	}

	if L := len(r.slice); L > 0 {
		r.client.observer.Failed(L)
	}

	err := r.err
	if err == nil {
		r.batch.ACK()
		return
	}

	r.batch.RetryEvents(r.slice)
	r.client.log.Errorf("Failed to publish events caused by: %+v", err)
}
