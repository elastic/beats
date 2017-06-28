package logstash

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/go-lumber/client/v2"
)

type syncClient struct {
	*transport.Client
	client *v2.SyncClient
	win    window
	ttl    time.Duration
	ticker *time.Ticker
}

func newSyncClient(conn *transport.Client, config *Config) (*syncClient, error) {
	c := &syncClient{}
	c.Client = conn
	c.ttl = config.TTL
	c.win.init(defaultStartMaxWindowSize, config.BulkMaxSize)
	if c.ttl > 0 {
		c.ticker = time.NewTicker(c.ttl)
	}

	var err error
	enc := makeLogstashEventEncoder(config.Index)
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
		logp.Err("error closing connection to logstash: %s, reconnecting...", err)
	}
	return c.Client.Connect()
}

func (c *syncClient) Publish(batch publisher.Batch) error {
	events := batch.Events()
	if len(events) == 0 {
		batch.ACK()
		return nil
	}

	publishEventsCallCount.Add(1)
	totalNumberOfEvents := int64(len(events))

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
				c.win.windowSize = int32(defaultStartMaxWindowSize)
			default:
			}
		}

		n, err := c.publishWindowed(events)
		events = events[n:]

		debugf("%v events out of %v events sent to logstash. Continue sending",
			n, len(events))

		if err != nil {
			// return batch to pipeline before reporting/counting error
			batch.RetryEvents(events)

			c.win.shrinkWindow()
			_ = c.Close()

			logp.Err("Failed to publish events caused by: %v", err)

			rest := int64(len(events))
			acked := totalNumberOfEvents - rest

			eventsNotAcked.Add(rest)
			ackedEvents.Add(acked)
			outputs.AckedEvents.Add(acked)

			return err
		}
	}

	batch.ACK()
	ackedEvents.Add(totalNumberOfEvents)
	outputs.AckedEvents.Add(totalNumberOfEvents)
	return nil
}

func (c *syncClient) publishWindowed(events []publisher.Event) (int, error) {
	batchSize := len(events)
	windowSize := c.win.get()
	debugf("Try to publish %v events to logstash with window size %v",
		batchSize, windowSize)

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
