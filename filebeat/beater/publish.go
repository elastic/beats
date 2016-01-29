package beater

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type logPublisher interface {
	Start()
	Stop()
}

type syncLogPublisher struct {
	client  publisher.Client
	in, out chan []*input.FileEvent

	done chan struct{}
	wg   sync.WaitGroup
}

type asyncLogPublisher struct {
	client  publisher.Client
	in, out chan []*input.FileEvent

	// list of in-flight batches
	active  batchList
	failing bool

	done chan struct{}
	wg   sync.WaitGroup
}

// eventsBatch is used to store sorted list of actively published log lines.
// Implements `outputs.Signalerr` interface for marking batch as finished
type eventsBatch struct {
	next   *eventsBatch
	flag   int32
	events []*input.FileEvent
}

type batchList struct {
	head, tail *eventsBatch
}

const (
	defaultGCTimeout = 1 * time.Second
)

const (
	batchSuccess int32 = 1
	batchFailed  int32 = 2
)

func newPublisher(
	async bool,
	in, out chan []*input.FileEvent,
	client publisher.Client,
) logPublisher {
	if async {
		return newAsyncLogPublisher(in, out, client)
	}
	return newSyncLogPublisher(in, out, client)
}

func newSyncLogPublisher(
	in, out chan []*input.FileEvent,
	client publisher.Client,
) *syncLogPublisher {
	return &syncLogPublisher{
		in:     in,
		out:    out,
		client: client,
		done:   make(chan struct{}),
	}
}

func (p *syncLogPublisher) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		logp.Info("Start sending events to output")

		for {
			var events []*input.FileEvent
			select {
			case <-p.done:
				return
			case events = <-p.in:
			}

			pubEvents := make([]common.MapStr, 0, len(events))
			for _, event := range events {
				pubEvents = append(pubEvents, event.ToMapStr())
			}

			p.client.PublishEvents(pubEvents, publisher.Sync, publisher.Guaranteed)
			logp.Info("Events sent: %d", len(events))

			// Tell the registrar that we've successfully sent these events
			select {
			case <-p.done:
				return
			case p.out <- events:
			}
		}
	}()
}

func (p *syncLogPublisher) Stop() {
	close(p.done)
	p.wg.Wait()
}

func newAsyncLogPublisher(
	in, out chan []*input.FileEvent,
	client publisher.Client,
) *asyncLogPublisher {
	return &asyncLogPublisher{
		in:     in,
		out:    out,
		client: client,
		done:   make(chan struct{}),
	}
}

func (p *asyncLogPublisher) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		logp.Info("Start sending events to output")

		// short gc timer, in case no logs are received from spooler the queued
		// bulkEvents can still be cleaned up and forwarded to the registrar
		ticker := time.NewTicker(defaultGCTimeout)

		for {
			select {
			case <-p.done:
				return
			case events := <-p.in:
				pubEvents := make([]common.MapStr, len(events))
				for i, event := range events {
					pubEvents[i] = event.ToMapStr()
				}

				batch := &eventsBatch{
					flag:   0,
					events: events,
				}
				p.client.PublishEvents(pubEvents,
					publisher.Signal(batch), publisher.Guaranteed)

				p.active.append(batch)
			case <-ticker.C:
			}

			p.collect()
		}
	}()
}

func (p *asyncLogPublisher) Stop() {
	close(p.done)
	p.wg.Wait()
}

// collect collects finished bulk-Events in order and forward processed batches
// to registrar. Reports to registrar are guaranteed to be in same order
// as bulk-Events have been received by the spooler
func (p *asyncLogPublisher) collect() bool {
	for batch := p.active.head; batch != nil; batch = batch.next {
		state := atomic.LoadInt32(&batch.flag)
		if state == 0 && !p.failing {
			break
		}

		// remove batch from active list
		p.active.head = batch.next
		if batch.next == nil {
			p.active.tail = nil
		}

		// Batches get marked as failed, if publisher pipeline is shutting down
		// In this case we do not want to send any more batches to the registrar
		if state == batchFailed {
			p.failing = true
		}

		if p.failing {
			// if in failing state keep cleaning up queue
			continue
		}

		// Tell the registrar that we've successfully sent these events
		select {
		case <-p.done:
			return false
		case p.out <- batch.events:
		}
	}
	return true
}

func (b *eventsBatch) Completed() { atomic.StoreInt32(&b.flag, batchSuccess) }
func (b *eventsBatch) Failed()    { atomic.StoreInt32(&b.flag, batchFailed) }

func (l *batchList) append(b *eventsBatch) {
	if l.head == nil {
		l.head = b
	} else {
		l.tail.next = b
	}
	b.next = nil
	l.tail = b
}
