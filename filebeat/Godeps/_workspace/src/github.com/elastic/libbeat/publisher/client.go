package publisher

import "github.com/elastic/libbeat/common"

// Client is used by beats to publish new events.
type Client interface {
	// PublishEvent publishes one event with given options. If Confirm option is set,
	// PublishEvent will block until output plugins report success or failure state
	// being returned by this method.
	PublishEvent(event common.MapStr, opts ...ClientOption) bool

	// PublishEvents publishes multiple events with given options. If Confirm
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
type ClientOption func(option *publishOptions)

// Confirm option will block the event publisher until event has been send and ACKed
// by output plugin or fail is reported.
func Confirm(options *publishOptions) {
	options.confirm = true
}

// Sync option will block the event publisher until an event has been ACKed by
// the output plugin. If output plugin signals failure, the client will retry
// until success is signaled.
func Sync(options *publishOptions) {
	options.confirm = true
	options.sync = true
}

func (c *client) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	options, client := c.getClient(opts)
	return client.PublishEvent(&context{publishOptions: options}, event)
}

func (c *client) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	options, client := c.getClient(opts)
	return client.PublishEvents(&context{publishOptions: options}, events)
}

func (c *client) getClient(opts []ClientOption) (publishOptions, eventPublisher) {
	debug("send event")
	options := publishOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.confirm {
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
