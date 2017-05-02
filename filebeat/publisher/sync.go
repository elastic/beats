package publisher

import (
	"sync"

	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type syncLogPublisher struct {
	pub    publisher.Publisher
	client publisher.Client
	in     chan []*util.Data
	out    SuccessLogger

	done chan struct{}
	wg   sync.WaitGroup
}

func newSyncLogPublisher(
	in chan []*util.Data,
	out SuccessLogger,
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
		defer logp.Debug("publisher", "Shutting down sync publisher")

		for {
			err := p.Publish()
			if err != nil {
				return
			}
		}
	}()
}

func (p *syncLogPublisher) Publish() error {
	var events []*util.Data
	select {
	case <-p.done:
		return sigPublisherStop
	case events = <-p.in:
	}

	dataEvents, meta := getDataEvents(events)
	ok := p.client.PublishEvents(dataEvents, publisher.Sync, publisher.Guaranteed,
		publisher.MetadataBatch(meta))
	if !ok {
		// PublishEvents will only returns false, if p.client has been closed.
		return sigPublisherStop
	}

	// TODO: move counter into logger?
	logp.Debug("publish", "Events sent: %d", len(events))
	eventsSent.Add(int64(len(events)))

	// Tell the logger that we've successfully sent these events
	ok = p.out.Published(events)
	if !ok {
		// stop publisher if successfully send events can not be logged anymore.
		return sigPublisherStop
	}
	return nil
}

func (p *syncLogPublisher) Stop() {
	p.client.Close()
	close(p.done)
	p.wg.Wait()
}
