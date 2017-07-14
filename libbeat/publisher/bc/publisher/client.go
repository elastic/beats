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
		canceler:  op.NewCanceler(),
		publisher: pub,
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
		datum:   makeEvent(event, metadata),
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
		metadata := metadataAll
		if meta != nil {
			metadata = meta[i]
		}
		data = append(data, makeEvent(event, metadata))
	}

	if len(data) == 0 {
		logp.Debug("filter", "No events to publish")
		return true
	}

	publishedEvents.Add(int64(len(data)))
	return pipeline.publish(message{client: c, context: ctx, data: data})
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
	if logp.IsDebug("publish") {
		logp.Debug("publish", "Publish: %s", fields.StringToPrint())
	}

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
