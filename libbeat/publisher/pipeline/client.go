package pipeline

import (
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

// client connects a beat with the processors and pipeline queue.
//
// TODO: All ackers currently drop any late incoming ACK. Some beats still might
//       be interested in handling/waiting for event ACKs more globally
//       -> add support for not dropping pending ACKs
type client struct {
	pipeline   *Pipeline
	processors beat.Processor
	producer   queue.Producer
	mutex      sync.Mutex
	acker      acker

	eventFlags   publisher.EventFlags
	canDrop      bool
	reportEvents bool

	isOpen atomic.Bool

	eventer beat.ClientEventer
}

func (c *client) PublishAll(events []beat.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, e := range events {
		c.publish(e)
	}
}

func (c *client) Publish(e beat.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.publish(e)
}

func (c *client) publish(e beat.Event) {
	var (
		event   = &e
		publish = true
		log     = c.pipeline.logger
	)

	c.onNewEvent()

	if !c.isOpen.Load() {
		// client is closing down -> report event as dropped and return
		c.onDroppedOnPublish(e)
		return
	}

	if c.processors != nil {
		var err error

		event, err = c.processors.Run(event)
		publish = event != nil
		if err != nil {
			// TODO: introduce dead-letter queue?

			log.Errorf("Failed to publish event: %v", err)
		}
	}

	if event != nil {
		e = *event
	}

	open := c.acker.addEvent(e, publish)
	if !open {
		// client is closing down -> report event as dropped and return
		c.onDroppedOnPublish(e)
		return
	}

	if !publish {
		c.onFilteredOut(e)
		return
	}

	e = *event
	pubEvent := publisher.Event{
		Content: e,
		Flags:   c.eventFlags,
	}

	if c.reportEvents {
		c.pipeline.waitCloser.inc()
	}

	var published bool
	if c.canDrop {
		published = c.producer.TryPublish(pubEvent)
	} else {
		published = c.producer.Publish(pubEvent)
	}

	if published {
		c.onPublished()
	} else {
		c.onDroppedOnPublish(e)
		if c.reportEvents {
			c.pipeline.waitCloser.dec(1)
		}
	}
}

func (c *client) Close() error {
	// first stop ack handling. ACK handler might block (with timeout), waiting
	// for pending events to be ACKed.

	log := c.pipeline.logger

	if !c.isOpen.Swap(false) {
		return nil // closed or already closing
	}

	c.onClosing()

	log.Debug("client: closing acker")
	c.acker.close()
	log.Debug("client: done closing acker")

	// finally disconnect client from broker
	n := c.producer.Cancel()
	log.Debugf("client: cancelled %v events", n)

	if c.reportEvents {
		log.Debugf("client: remove client events")
		if n > 0 {
			c.pipeline.waitCloser.dec(n)
		}
	}

	c.onClosed()
	return nil
}

func (c *client) onClosing() {
	c.pipeline.observer.clientClosing()
	if c.eventer != nil {
		c.eventer.Closing()
	}
}

func (c *client) onClosed() {
	c.pipeline.observer.clientClosed()
	if c.eventer != nil {
		c.eventer.Closed()
	}
}

func (c *client) onNewEvent() {
	c.pipeline.observer.newEvent()
}

func (c *client) onPublished() {
	c.pipeline.observer.publishedEvent()
	if c.eventer != nil {
		c.eventer.Published()
	}
}

func (c *client) onFilteredOut(e beat.Event) {
	c.pipeline.observer.filteredEvent()
	if c.eventer != nil {
		c.eventer.FilteredOut(e)
	}
}

func (c *client) onDroppedOnPublish(e beat.Event) {
	c.pipeline.observer.failedPublishEvent()
	if c.eventer != nil {
		c.eventer.DroppedOnPublish(e)
	}
}
