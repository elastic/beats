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
//
// The publish methods add fields that are common to all events. Both methods
// add the 'beat' field that contains name and hostname. Also they add 'tags'
// and 'fields'.
//
// Event publishers can override the default index for an event by adding a
// 'beat' field whose value is a common.MapStr that contains an 'index' field
// specifying the destination index.
//
//  event := common.MapStr{
//      // Setting a custom index for a single event.
//      "beat": common.MapStr{"index": "custom-index"},
//  }
//
// Event publishers can add fields and tags to an event. The fields will take
// precedence over the global fields defined in the shipper configuration.
//
//  event := common.MapStr{
//      // Add custom fields to the root of the event.
//      common.EventMetadataKey: common.EventMetadata{
//          UnderRoot: true,
//          Fields:    common.MapStr{"env": "production"}
//      }
//  }
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
	publisher           *PublisherType
	beatMeta            common.MapStr        // Beat metadata that is added to all events.
	globalEventMetadata common.EventMetadata // Fields and tags that are added to all events.
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
		globalEventMetadata: pub.globalEventMetadata,
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

// annotateEvent adds fields that are common to all events. This adds the 'beat'
// field that contains name and hostname. It also adds 'tags' and 'fields'. See
// the documentation for Client for more information.
func (c *client) annotateEvent(event common.MapStr) {
	// Allow an event to override the destination index for an event by setting
	// beat.index in an event.
	beatMeta := c.beatMeta
	if beatIfc, ok := event["beat"]; ok {
		ms, ok := beatIfc.(common.MapStr)
		if ok {
			// Copy beatMeta so the defaults are not changed.
			beatMeta = common.MapStrUnion(beatMeta, ms)
		}
	}
	event["beat"] = beatMeta

	// Add the global tags and fields defined under shipper.
	common.AddTags(event, c.globalEventMetadata.Tags)
	common.MergeFields(event, c.globalEventMetadata.Fields, c.globalEventMetadata.FieldsUnderRoot)

	// Add the event specific fields last so that they precedence over globals.
	if metaIfc, ok := event[common.EventMetadataKey]; ok {
		eventMetadata, ok := metaIfc.(common.EventMetadata)
		if ok {
			common.AddTags(event, eventMetadata.Tags)
			common.MergeFields(event, eventMetadata.Fields, eventMetadata.FieldsUnderRoot)
		}
		delete(event, common.EventMetadataKey)
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
