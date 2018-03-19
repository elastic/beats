package logstash

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/go-lumber/client/v2"
)

type syncClient struct {
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
	c := &syncClient{
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
	enc := makeLogstashEventEncoder(beat, config.Index)
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
	logp.Debug("logstash", "connect")
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
	logp.Debug("logstash", "close connection")
	return c.Client.Close()
}

func (c *syncClient) reconnect() error {
	if err := c.Client.Close(); err != nil {
		logp.Err("error closing connection to logstash host %s: %s, reconnecting...", c.Host(), err)
	}
	return c.Client.Connect()
}

func (c *syncClient) Publish(batch publisher.Batch) error {
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

		if c.win == nil {
			n, err = c.sendEvents(events)
		} else {
			n, err = c.publishWindowed(events)
		}

		debugf("%v events out of %v events sent to logstash host %s. Continue sending",
			n, len(events), c.Host())

		events = events[n:]
		st.Acked(n)
		if err != nil {
			// return batch to pipeline before reporting/counting error
			batch.RetryEvents(events)

			if c.win != nil {
				c.win.shrinkWindow()
			}
			_ = c.Close()

			logp.Err("Failed to publish events caused by: %v", err)

			rest := len(events)
			st.Failed(rest)

			return err
		}
	}

	batch.ACK()
	return nil
}

func (c *syncClient) publishWindowed(events []publisher.Event) (int, error) {
	batchSize := len(events)
	windowSize := c.win.get()
	debugf("Try to publish %v events to logstash host %s with window size %v",
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
