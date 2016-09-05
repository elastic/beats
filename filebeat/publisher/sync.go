package publisher

import (
	"errors"
	"sync"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type syncLogPublisher struct {
	pub     publisher.Publisher
	client  publisher.Client
	in, out chan []*input.Event

	done chan struct{}
	wg   sync.WaitGroup
}

func newSyncLogPublisher(
	in, out chan []*input.Event,
	pub publisher.Publisher,
) *syncLogPublisher {
	return &syncLogPublisher{
		in:   in,
		out:  out,
		pub:  pub,
		done: make(chan struct{}),
	}
}

func (p *syncLogPublisher) Start() {
	p.client = p.pub.Connect()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		logp.Info("Start sending events to output")

		for {
			err := p.Publish()
			if err != nil {
				logp.Debug("publisher", "Shutting down sync publisher")
				return
			}
		}
	}()
}

func (p *syncLogPublisher) Publish() error {
	var events []*input.Event
	select {
	case <-p.done:
		return errors.New("publishing was stopped")
	case events = <-p.in:
	}

	ok := p.client.PublishEvents(getDataEvents(events), publisher.Sync, publisher.Guaranteed)
	if !ok {
		// PublishEvents will only returns false, if p.client has been closed.
		return errors.New("publisher didn't published events")
	}

	logp.Debug("publish", "Events sent: %d", len(events))
	eventsSent.Add(int64(len(events)))

	// Tell the registrar that we've successfully sent these events
	select {
	case <-p.done:
		return errors.New("publishing was stopped")
	case p.out <- events:
	}

	return nil
}

func (p *syncLogPublisher) Stop() {
	p.client.Close()
	close(p.done)
	p.wg.Wait()
}
