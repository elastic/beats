package publisher

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/publisher/beat"
)

// Metrics that can retrieved through the expvar web interface.
var (
	publishedEvents = monitoring.NewInt(nil, "publisher.events.count")
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

	publisher *BeatPublisher
	sync      *syncClient
	async     *asyncClient

	beatMeta            common.MapStr        // Beat metadata that is added to all events.
	globalEventMetadata common.EventMetadata // Fields and tags that are added to all events.
}

type message struct {
	client  *client
	context Context
	datum   beat.Event
	data    []beat.Event
}

type sender interface {
	publish(message) bool
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
	c.publisher.numClients.Dec()
	return nil
}

func (c *client) PublishEvent(event common.MapStr, opts ...ClientOption) bool {
	c.annotateEvent(event)

	publishEvent := c.filterEvent(event)
	if publishEvent == nil {
		return false
	}

	var metadata common.MapStr
	meta, ctx, pipeline, err := c.getPipeline(opts)
	if err != nil {
		panic(err)
	}

	if len(meta) != 0 {
		if len(meta) != 1 {
			logp.Debug("publish", "too many metadata, pick first")
		}
		metadata = meta[0]
	}

	publishedEvents.Add(1)
	return pipeline.publish(message{
		client:  c,
		context: ctx,
		datum:   makeEvent(*publishEvent, metadata),
	})
}

func (c *client) PublishEvents(events []common.MapStr, opts ...ClientOption) bool {
	var metadataAll common.MapStr
	meta, ctx, pipeline, err := c.getPipeline(opts)
	if err != nil {
		panic(err)
	}

	if len(meta) != 0 && len(events) != len(meta) {
		if len(meta) != 1 {
			logp.Debug("publish",
				"Number of metadata elements does not match number of events => dropping metadata")
			meta = nil
		} else {
			metadataAll = meta[0]
			meta = nil
		}
	}

	data := make([]beat.Event, 0, len(events))
	for i, event := range events {
		c.annotateEvent(event)

		publishEvent := c.filterEvent(event)
		if publishEvent == nil {
			continue
		}

		metadata := metadataAll
		if meta != nil {
			metadata = meta[i]
		}
		data = append(data, makeEvent(*publishEvent, metadata))
	}

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
	publishEvent := c.publisher.processors.Run(event)
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

func (c *client) getPipeline(opts []ClientOption) ([]common.MapStr, Context, sender, error) {

	var err error
	values, ctx := MakeContext(opts)

	if ctx.Sync {
		if c.sync == nil {
			c.sync, err = newSyncClient(c.publisher, c.canceler.Done())
			if err != nil {
				return nil, ctx, nil, err
			}
		}
		return values, ctx, c.sync, nil
	}

	if c.async == nil {
		c.async, err = newAsyncClient(c.publisher, c.canceler.Done())
		if err != nil {
			return nil, ctx, nil, err
		}
	}
	return values, ctx, c.async, nil
}

func MakeContext(opts []ClientOption) ([]common.MapStr, Context) {
	var ctx Context
	var meta []common.MapStr
	for _, opt := range opts {
		var m []common.MapStr
		m, ctx = opt(ctx)
		if m != nil {
			if meta == nil {
				meta = m
			} else {
				meta = append(meta, m...)
			}
		}
	}
	return meta, ctx
}

func makeEvent(fields common.MapStr, meta common.MapStr) beat.Event {
	var ts time.Time
	switch value := fields["@timestamp"].(type) {
	case time.Time:
		ts = value
	case common.Time:
		ts = time.Time(value)
	default:
		ts = time.Now()
	}
	delete(fields, "@timestamp")

	return beat.Event{
		Timestamp: ts,
		Meta:      meta,
		Fields:    fields,
	}
}
