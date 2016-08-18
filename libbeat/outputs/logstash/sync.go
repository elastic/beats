package logstash

import (
	"time"

	"github.com/elastic/go-lumber/client/v2"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
)

type client struct {
	*transport.Client
	client *v2.SyncClient
	win    window
}

func newLumberjackClient(
	conn *transport.Client,
	compressLevel int,
	maxWindowSize int,
	timeout time.Duration,
	beat string,
) (*client, error) {
	c := &client{}
	c.Client = conn
	c.win.init(defaultStartMaxWindowSize, maxWindowSize)

	enc, err := makeLogstashEventEncoder(beat)
	if err != nil {
		return nil, err
	}

	cl, err := v2.NewSyncClientWithConn(conn,
		v2.JSONEncoder(enc),
		v2.Timeout(timeout),
		v2.CompressionLevel(compressLevel))
	if err != nil {
		return nil, err
	}

	c.client = cl
	return c, nil
}

func (c *client) Connect(timeout time.Duration) error {
	logp.Debug("logstash", "connect")
	return c.Client.Connect()
}

func (c *client) Close() error {
	logp.Debug("logstash", "close connection")
	return c.Client.Close()
}

func (c *client) PublishEvent(data outputs.Data) error {
	_, err := c.PublishEvents([]outputs.Data{data})
	return err
}

// PublishEvents sends all events to logstash. On error a slice with all events
// not published or confirmed to be processed by logstash will be returned.
func (l *client) PublishEvents(
	data []outputs.Data,
) ([]outputs.Data, error) {
	publishEventsCallCount.Add(1)
	totalNumberOfEvents := len(data)
	for len(data) > 0 {
		n, err := l.publishWindowed(data)

		debug("%v events out of %v events sent to logstash. Continue sending",
			n, len(data))

		data = data[n:]
		if err != nil {
			l.win.shrinkWindow()
			_ = l.Close()

			logp.Err("Failed to publish events caused by: %v", err)

			eventsNotAcked.Add(int64(len(data)))
			ackedEvents.Add(int64(totalNumberOfEvents - len(data)))
			return data, err
		}
	}
	ackedEvents.Add(int64(totalNumberOfEvents))
	return nil, nil
}

// publishWindowed published events with current maximum window size to logstash
// returning the total number of events sent (due to window size, or acks until
// failure).
func (c *client) publishWindowed(data []outputs.Data) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	batchSize := len(data)
	windowSize := c.win.get()
	debug("Try to publish %v events to logstash with window size %v",
		batchSize, windowSize)

	// prepare message payload
	if batchSize > windowSize {
		data = data[:windowSize]
	}

	n, err := c.sendEvents(data)
	if err != nil {
		return n, err
	}

	c.win.tryGrowWindow(batchSize)
	return len(data), nil
}

func (c *client) sendEvents(data []outputs.Data) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	window := make([]interface{}, len(data))
	for i, d := range data {
		window[i] = d
	}
	return c.client.Send(window)
}
