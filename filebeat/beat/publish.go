package beat

import (
	"sync/atomic"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
)

type logPublisher struct {
	client  publisher.Client
	in, out chan []*input.FileEvent

	// list of in-flight batches
	active batchList
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

func (b *eventsBatch) Completed() { atomic.StoreInt32(&b.flag, 1) }
func (b *eventsBatch) Failed()    { atomic.StoreInt32(&b.flag, 1) }

func newLogPublisher(in, out chan []*input.FileEvent, client publisher.Client) *logPublisher {
	return &logPublisher{
		in:     in,
		out:    out,
		client: client,
	}
}

func (pub *logPublisher) start() {
	go func() {
		logp.Info("Start sending events to output")

		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case events := <-pub.in:
				pubEvents := make([]common.MapStr, len(events))
				for i, event := range events {
					pubEvents[i] = event.ToMapStr()
				}

				batch := &eventsBatch{
					flag:   0,
					events: events,
				}
				pub.client.PublishEvents(pubEvents,
					publisher.Signal(batch), publisher.Guaranteed)

				pub.active.append(batch)
			case <-ticker.C:
			}

			// forward processed batches to registrar
			for batch := pub.active.head; batch != nil; batch = batch.next {
				if atomic.LoadInt32(&batch.flag) == 0 {
					break
				}

				// remove batch from active list
				pub.active.head = batch.next
				if batch.next == nil {
					pub.active.tail = nil
				}

				// Tell the registrar that we've successfully sent these events
				logp.Info("Events sent: %d", len(batch.events))
				pub.out <- batch.events
			}
		}
	}()
}

func (l *batchList) append(b *eventsBatch) {
	if l.head == nil {
		l.head = b
	} else {
		l.tail.next = b
	}
	b.next = nil
	l.tail = b
}
