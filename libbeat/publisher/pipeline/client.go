package pipeline

import (
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/beat"
	"github.com/elastic/beats/libbeat/publisher/broker"
)

// client connects a beat with the processors and pipeline broker.
//
// TODO: All ackers currently drop any late incoming ACK. Some beats still might
//       be interested in handling/waiting for event ACKs more globally
//       -> add support for not dropping pending ACKs
type client struct {
	// active connection to broker
	pipeline   *Pipeline
	processors beat.Processor
	producer   broker.Producer
	acker      acker

	eventFlags   publisher.EventFlags
	canDrop      bool
	cancelEvents bool
	reportEvents bool
}

func (c *client) PublishAll(events []beat.Event) {
	for _, e := range events {
		c.Publish(e)
	}
}

func (c *client) Publish(e beat.Event) {
	var (
		event   = &e
		publish = true
	)

	if c.processors != nil {
		var err error

		event, err = c.processors.Run(event)
		publish = event != nil
		if err != nil {
			// TODO: introduce dead-letter queue?

			log := c.pipeline.logger
			log.Errf("Failed to publish event: %v", err)
		}
	}

	if event != nil {
		e = *event
	}

	c.acker.addEvent(e, publish)
	if !publish {
		return
	}

	e = *event
	pubEvent := publisher.Event{
		Content: e,
		Flags:   c.eventFlags,
	}

	dropped := false
	if c.canDrop {
		if c.reportEvents {
			c.pipeline.events.Add(1)
		}
		dropped = !c.producer.TryPublish(pubEvent)
		if dropped && c.reportEvents {
			c.pipeline.activeEventsDone(1)
		}
	} else {
		if c.reportEvents {
			c.pipeline.activeEventsAdd(1)
		}
		c.producer.Publish(pubEvent)
	}
}

func (c *client) Close() error {
	// first stop ack handling. ACK handler might block (with timeout), waiting
	// for pending events to be ACKed.

	log := c.pipeline.logger

	log.Debug("client: closing acker")
	c.acker.close()
	log.Debug("client: done closing acker")

	// finally disconnect client from broker
	if c.cancelEvents {

		n := c.producer.Cancel()
		log.Debugf("client: cancelled %v events", n)

		if c.reportEvents {
			log.Debugf("client: remove client events")
			c.pipeline.activeEventsDone(n)
		}
	}

	return nil
}
