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
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport"
	v2 "github.com/elastic/go-lumber/client/v2"
)

type syncClient struct {
	log *logp.Logger
	*transport.Client
	client   *v2.SyncClient
	observer outputs.Observer
	win      *window
	ttl      time.Duration
	ticker   *time.Ticker
}

func newSyncClient(
	beat beat.Info,
	conn *transport.Client,
	observer outputs.Observer,
	config *Config,
) (*syncClient, error) {
	log := logp.NewLogger("logstash")
	c := &syncClient{
		log:      log,
		Client:   conn,
		observer: observer,
		ttl:      config.TTL,
	}

	if config.SlowStart {
		c.win = newWindower(defaultStartMaxWindowSize, config.BulkMaxSize)
	}
	if c.ttl > 0 {
		c.ticker = time.NewTicker(c.ttl)
	}

	var err error
	enc := makeLogstashEventEncoder(log, beat, config.EscapeHTML, config.Index)
	c.client, err = v2.NewSyncClientWithConn(conn,
		v2.JSONEncoder(enc),
		v2.Timeout(config.Timeout),
		v2.CompressionLevel(config.CompressionLevel),
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *syncClient) Connect() error {
	c.log.Debug("connect")
	err := c.Client.Connect()
	if err != nil {
		return err
	}

	if c.ticker != nil {
		c.ticker = time.NewTicker(c.ttl)
	}
	return nil
}

func (c *syncClient) Close() error {
	if c.ticker != nil {
		c.ticker.Stop()
	}
	c.log.Debug("close connection")
	return c.Client.Close()
}

func (c *syncClient) reconnect() error {
	if err := c.Client.Close(); err != nil {
		c.log.Errorf("error closing connection to logstash host %s: %+v, reconnecting...", c.Host(), err)
	}
	return c.Client.Connect()
}

func (c *syncClient) Publish(_ context.Context, batch publisher.Batch) error {
	events := batch.Events()
	st := c.observer

	st.NewBatch(len(events))

	if len(events) == 0 {
		batch.ACK()
		return nil
	}

	for len(events) > 0 {

		// check if we need to reconnect
		if c.ticker != nil {
			select {
			case <-c.ticker.C:
				if err := c.reconnect(); err != nil {
					batch.Retry()
					return err
				}

				// reset window size on reconnect
				if c.win != nil {
					c.win.windowSize = int32(defaultStartMaxWindowSize)
				}
			default:
			}
		}

		var (
			n   int
			err error
		)

		begin := time.Now()
		if c.win == nil {
			n, err = c.sendEvents(events)
		} else {
			n, err = c.publishWindowed(events)
		}
		took := time.Since(begin)
		st.ReportLatency(took)
		c.log.Debugf("%v events out of %v events sent to logstash host %s. Continue sending",
			n, len(events), c.Host())

		events = events[n:]
		st.AckedEvents(n)
		if err != nil {
			// return batch to pipeline before reporting/counting error
			batch.RetryEvents(events)

			if c.win != nil {
				c.win.shrinkWindow()
			}
			_ = c.Close()

			c.log.Errorf("Failed to publish events caused by: %+v", err)

			rest := len(events)
			st.RetryableErrors(rest)

			return err
		}

	}

	batch.ACK()
	return nil
}

func (c *syncClient) publishWindowed(events []publisher.Event) (int, error) {
	batchSize := len(events)
	windowSize := c.win.get()
	c.log.Debugf("Try to publish %v events to logstash host %s with window size %v",
		batchSize, c.Host(), windowSize)

	// prepare message payload
	if batchSize > windowSize {
		events = events[:windowSize]
	}

	n, err := c.sendEvents(events)
	if err != nil {
		return n, err
	}

	c.win.tryGrowWindow(batchSize)
	return n, nil
}

func (c *syncClient) sendEvents(events []publisher.Event) (int, error) {
	window := make([]interface{}, len(events))
	for i := range events {
		window[i] = &events[i].Content
	}
	return c.client.Send(window)
}
