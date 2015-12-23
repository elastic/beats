package publisher

import (
	"expvar"

	"github.com/elastic/beats/libbeat/common"
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
type ClientOption func(option publishOptions) publishOptions

// Guaranteed option will retry publishing the event, until send attempt have
// been ACKed by output plugin.
func Guaranteed(o publishOptions) publishOptions {
	o.guaranteed = true
	return o
}

// Sync option will block the event publisher until an event has been ACKed by
// the output plugin or failed.
func Sync(o publishOptions) publishOptions {
	o.sync = true
	return o
}

func (c *client) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	options, client := c.getClient(opts)
	publishedEvents.Add(1)
	return client.PublishEvent(context{publishOptions: options}, event)
}

func (c *client) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	options, client := c.getClient(opts)
	publishedEvents.Add(int64(len(events)))
	return client.PublishEvents(context{publishOptions: options}, events)
}

func (c *client) getClient(opts []ClientOption) (publishOptions, eventPublisher) {
	var options publishOptions
	for _, opt := range opts {
		options = opt(options)
	}

	if options.guaranteed {
		return options, c.publisher.syncPublisher.client()
	}
	return options, c.publisher.asyncPublisher.client()
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
