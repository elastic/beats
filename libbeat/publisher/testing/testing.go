package testing

// ChanClient implements Client interface, forwarding published events to some
import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

type TestPublisher struct {
	client publisher.Client
}

// given channel only.
type ChanClient struct {
	done    chan struct{}
	Channel chan PublishMessage

	recvBuf []common.MapStr
}

type PublishMessage struct {
	Context publisher.Context
	Events  []common.MapStr
}

func PublisherWithClient(client publisher.Client) publisher.Publisher {
	return &TestPublisher{client}
}

func (pub *TestPublisher) Connect() publisher.Client {
	return pub.client
}

func NewChanClient(bufSize int) *ChanClient {
	return NewChanClientWith(make(chan PublishMessage, bufSize))
}

func NewChanClientWith(ch chan PublishMessage) *ChanClient {
	if ch == nil {
		ch = make(chan PublishMessage, 1)
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
func (c *ChanClient) PublishEvent(event common.MapStr, opts ...publisher.ClientOption) bool {
	return c.PublishEvents([]common.MapStr{event}, opts...)
}

// PublishEvents publishes all event on the configured channel. Options will be ignored.
// Always returns true.
func (c *ChanClient) PublishEvents(events []common.MapStr, opts ...publisher.ClientOption) bool {
	msg := PublishMessage{publisher.MakeContext(opts), events}
	select {
	case <-c.done:
		return false
	case c.Channel <- msg:
		return true
	}
}

func (c *ChanClient) ReceiveEvent() common.MapStr {
	if len(c.recvBuf) > 0 {
		evt := c.recvBuf[0]
		c.recvBuf = c.recvBuf[1:]
		return evt
	}

	msg := <-c.Channel
	c.recvBuf = msg.Events
	return c.ReceiveEvent()
}

func (c *ChanClient) ReceiveEvents() []common.MapStr {
	if len(c.recvBuf) > 0 {
		return c.recvBuf
	}

	msg := <-c.Channel
	return msg.Events
}
