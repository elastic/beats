package publisher

import (
	"expvar"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
)

// Metrics that can retrieved through the expvar web interface.
var (
	publishedEvents = expvar.NewInt("libbeatPublishedEvents")
)

// Client is used by beats to publish new events.
type Client interface {
	// PublishEvent publishes one event with given options. If Sync option is set,
	// PublishEvent will block until output plugins report success or failure state
	// being returned by this method.
	PublishEvent(event common.MapStr, opts ...ClientOption) bool

	// PublishEvents publishes multiple events with given options. If Guaranteed
	// option is set, PublishEvent will block until output plugins report
	// success or failure state being returned by this method.
	PublishEvents(events []common.MapStr, opts ...ClientOption) bool
}

// ChanClient will forward all published events one by one to the given channel
type ChanClient struct {
	Channel chan common.MapStr
}

type client struct {
	publisher *PublisherType
}

// ClientOption allows API users to set additional options when publishing events.
type ClientOption func(option context) context

// Guaranteed option will retry publishing the event, until send attempt have
// been ACKed by output plugin.
func Guaranteed(o context) context {
	o.guaranteed = true
	return o
}

// Sync option will block the event publisher until an event has been ACKed by
// the output plugin or failed.
func Sync(o context) context {
	o.sync = true
	return o
}

func Signal(signaler outputs.Signaler) ClientOption {
	return func(ctx context) context {
		if ctx.signal == nil {
			ctx.signal = signaler
		} else {
			ctx.signal = outputs.NewCompositeSignaler(ctx.signal, signaler)
		}
		return ctx
	}
}

func (c *client) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	ctx, client := c.getClient(opts)
	publishedEvents.Add(1)
	return client.PublishEvent(ctx, event)
}

func (c *client) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	ctx, client := c.getClient(opts)
	publishedEvents.Add(int64(len(events)))
	return client.PublishEvents(ctx, events)
}

func (c *client) getClient(opts []ClientOption) (context, eventPublisher) {
	var ctx context
	for _, opt := range opts {
		ctx = opt(ctx)
	}

	if ctx.sync {
		return ctx, c.publisher.syncPublisher.client()
	}
	return ctx, c.publisher.asyncPublisher.client()
}

// PublishEvent will publish the event on the channel. Options will be ignored.
// Always returns true.
func (c ChanClient) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	c.Channel <- event
	return true
}

// PublishEvents publishes all event on the configured channel. Options will be ignored.
// Always returns true.
func (c ChanClient) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	for _, event := range events {
		c.Channel <- event
	}
	return true
}
