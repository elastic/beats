package publisher

import (
	"expvar"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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

type ExtChanClient struct {
	Channel chan PublishMessage
}

type PublishMessage struct {
	Context Context
	Events  []common.MapStr
}

type client struct {
	publisher *PublisherType

	beatMeta common.MapStr
	tags     []string
}

// ClientOption allows API users to set additional options when publishing events.
type ClientOption func(option Context) Context

// Guaranteed option will retry publishing the event, until send attempt have
// been ACKed by output plugin.
func Guaranteed(o Context) Context {
	o.Guaranteed = true
	return o
}

// Sync option will block the event publisher until an event has been ACKed by
// the output plugin or failed.
func Sync(o Context) Context {
	o.Sync = true
	return o
}

func Signal(signaler outputs.Signaler) ClientOption {
	return func(ctx Context) Context {
		if ctx.Signal == nil {
			ctx.Signal = signaler
		} else {
			ctx.Signal = outputs.NewCompositeSignaler(ctx.Signal, signaler)
		}
		return ctx
	}
}

func newClient(pub *PublisherType) *client {
	return &client{
		publisher: pub,
		beatMeta: common.MapStr{
			"name":     pub.name,
			"hostname": pub.hostname,
		},
		tags: pub.tags,
	}
}

func (c *client) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	c.annotateEvent(event)

	ctx, client := c.getClient(opts)
	publishedEvents.Add(1)
	return client.PublishEvent(ctx, event)
}

func (c *client) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	for _, event := range events {
		c.annotateEvent(event)
	}

	ctx, client := c.getClient(opts)
	publishedEvents.Add(int64(len(events)))
	return client.PublishEvents(ctx, events)
}

func (c *client) annotateEvent(event common.MapStr) {

	// Check if index was set dynamically
	if _, ok := event["beat"]; ok {
		beatTemp := event["beat"].(common.MapStr)
		if _, ok := beatTemp["index"]; ok {
			c.beatMeta["index"] = beatTemp["index"]
		}
	}

	event["beat"] = c.beatMeta
	if len(c.tags) > 0 {
		event["tags"] = c.tags
	}

	if logp.IsDebug("publish") {
		PrintPublishEvent(event)
	}
}

func (c *client) getClient(opts []ClientOption) (Context, eventPublisher) {
	ctx := makeContext(opts)
	if ctx.Sync {
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

// PublishEvent will publish the event on the channel. Options will be ignored.
// Always returns true.
func (c ExtChanClient) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	c.Channel <- PublishMessage{makeContext(opts), []common.MapStr{event}}
	return true
}

// PublishEvents publishes all event on the configured channel. Options will be ignored.
// Always returns true.
func (c ExtChanClient) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	c.Channel <- PublishMessage{makeContext(opts), events}
	return true
}

func makeContext(opts []ClientOption) Context {
	var ctx Context
	for _, opt := range opts {
		ctx = opt(ctx)
	}
	return ctx
}
