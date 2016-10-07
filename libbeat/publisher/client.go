package publisher

import (
	"errors"
	"expvar"
	"sync/atomic"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

// Metrics that can retrieved through the expvar web interface.
var (
	publishedEvents = expvar.NewInt("libbeat.publisher.published_events")
)

var (
	ErrClientClosed = errors.New("client closed")
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
	// Close disconnects the Client from the publisher pipeline.
	Close() error

	// PublishEvent publishes one event with given options. If Sync option is set,
	// PublishEvent will block until output plugins report success or failure state
	// being returned by this method.
	PublishEvent(event common.MapStr, opts ...ClientOption) bool

	// PublishEvents publishes multiple events with given options. If Guaranteed
	// option is set, PublishEvent will block until output plugins report
	// success or failure state being returned by this method.
	PublishEvents(events []common.MapStr, opts ...ClientOption) bool
}

type client struct {
	canceler *op.Canceler

	publisher           *BeatPublisher
	beatMeta            common.MapStr        // Beat metadata that is added to all events.
	globalEventMetadata common.EventMetadata // Fields and tags that are added to all events.
}

func newClient(pub *BeatPublisher) *client {
	c := &client{
		canceler: op.NewCanceler(),

		publisher: pub,
		beatMeta: common.MapStr{
			"name":     pub.name,
			"hostname": pub.hostname,
			"version":  pub.version,
		},
		globalEventMetadata: pub.globalEventMetadata,
	}
	return c
}

func (c *client) Close() error {
	if c == nil {
		return nil
	}

	c.canceler.Cancel()

	// atomic decrement clients counter
	atomic.AddUint32(&c.publisher.numClients, ^uint32(0))
	return nil
}

func (c *client) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	c.annotateEvent(event)

	publishEvent := c.filterEvent(event)
	if publishEvent == nil {
		return false
	}

	ctx, pipeline := c.getPipeline(opts)
	publishedEvents.Add(1)
	return pipeline.publish(message{
		client:  c,
		context: ctx,
		datum:   outputs.Data{Event: *publishEvent},
	})
}

func (c *client) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	data := make([]outputs.Data, 0, len(events))
	for _, event := range events {
		c.annotateEvent(event)

		publishEvent := c.filterEvent(event)
		if publishEvent != nil {
			data = append(data, outputs.Data{Event: *publishEvent})
		}
	}

	ctx, pipeline := c.getPipeline(opts)
	if len(data) == 0 {
		logp.Debug("filter", "No events to publish")
		return true
	}

	publishedEvents.Add(int64(len(data)))
	return pipeline.publish(message{client: c, context: ctx, data: data})
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

}

func (c *client) filterEvent(event common.MapStr) *common.MapStr {

	if event = common.ConvertToGenericEvent(event); event == nil {
		logp.Err("fail to convert to a generic event")
		return nil

	}

	// process the event by applying the configured actions
	publishEvent := c.publisher.Processors.Run(event)
	if publishEvent == nil {
		// the event is dropped
		logp.Debug("publish", "Drop event %s", event.StringToPrint())
		return nil
	}
	if logp.IsDebug("publish") {
		logp.Debug("publish", "Publish: %s", publishEvent.StringToPrint())
	}
	return &publishEvent
}

func (c *client) getPipeline(opts []ClientOption) (Context, pipeline) {
	ctx := MakeContext(opts)
	if ctx.Sync {
		return ctx, c.publisher.pipelines.sync
	}
	return ctx, c.publisher.pipelines.async
}

func MakeContext(opts []ClientOption) Context {
	var ctx Context
	for _, opt := range opts {
		ctx = opt(ctx)
	}
	return ctx
}
