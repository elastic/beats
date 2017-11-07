package testing

// ChanClient implements Client interface, forwarding published events to some
import (
	"github.com/elastic/beats/libbeat/beat"
)

type TestPublisher struct {
	client beat.Client
}

// given channel only.
type ChanClient struct {
	done    chan struct{}
	Channel chan beat.Event
}

func PublisherWithClient(client beat.Client) beat.Pipeline {
	return &TestPublisher{client}
}

func (pub *TestPublisher) Connect() (beat.Client, error) {
	return pub.client, nil
}

func (pub *TestPublisher) ConnectWith(_ beat.ClientConfig) (beat.Client, error) {
	return pub.client, nil
}

func (pub *TestPublisher) SetACKHandler(_ beat.PipelineACKHandler) error {
	panic("Not supported")
}

func NewChanClient(bufSize int) *ChanClient {
	return NewChanClientWith(make(chan beat.Event, bufSize))
}

func NewChanClientWith(ch chan beat.Event) *ChanClient {
	if ch == nil {
		ch = make(chan beat.Event, 1)
	}
	c := &ChanClient{
		done:    make(chan struct{}),
		Channel: ch,
	}
	return c
}

func (c *ChanClient) Close() error {
	close(c.done)
	return nil
}

// PublishEvent will publish the event on the channel. Options will be ignored.
// Always returns true.
func (c *ChanClient) Publish(event beat.Event) {
	select {
	case <-c.done:
	case c.Channel <- event:
	}
}

func (c *ChanClient) PublishAll(event []beat.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}

func (c *ChanClient) ReceiveEvent() beat.Event {
	return <-c.Channel
}
