package publisher

import (
	"sync"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type syncPublisher struct {
	pub    publisher.Publisher
	client publisher.Client
	in     chan []*input.Event
	logger Logger
	done   chan struct{}
	wg     sync.WaitGroup
}

func newSyncPublisher(
	in chan []*input.Event,
	logger Logger,
	pub publisher.Publisher,
) *syncPublisher {
	return &syncPublisher{
		in:     in,
		logger: logger,
		pub:    pub,
		done:   make(chan struct{}),
	}
}

func (p *syncPublisher) Start() {
	p.client = p.pub.Connect()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		logp.Info("Start sending events to output")
		defer logp.Debug("publisher", "Shutting down sync publisher")

		for {
			err := p.Publish()
			if err != nil {
				return
			}
		}
	}()
}

func (p *syncPublisher) Publish() error {
	var events []*input.Event
	select {
	case <-p.done:
		return sigPublisherStop
	case events = <-p.in:
	}

	ok := p.client.PublishEvents(getDataEvents(events), publisher.Sync, publisher.Guaranteed)
	if !ok {
		// PublishEvents will only returns false, if p.client has been closed.
		return sigPublisherStop
	}

	// TODO: move counter into logger?
	logp.Debug("publish", "Events sent: %d", len(events))
	eventsSent.Add(int64(len(events)))

	// Tell the logger that we've successfully sent these events
	ok = p.logger.Log(events)
	if !ok {
		// stop publisher if successfully send events can not be logged anymore.
		return sigPublisherStop
	}
	return nil
}

func (p *syncPublisher) Stop() {
	p.client.Close()
	close(p.done)
	p.wg.Wait()
}
