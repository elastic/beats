package pipeline

import (
	"errors"

	"github.com/elastic/beats/libbeat/beat"
)

type ackBuilder interface {
	createPipelineACKer(canDrop bool, sema *sema) acker
	createCountACKer(canDrop bool, sema *sema, fn func(int)) acker
	createEventACKer(canDrop bool, sema *sema, fn func([]interface{})) acker
}

type pipelineEmptyACK struct {
	pipeline *Pipeline
}

func (b *pipelineEmptyACK) createPipelineACKer(canDrop bool, sema *sema) acker {
	return nilACKer
}

func (b *pipelineEmptyACK) createCountACKer(canDrop bool, sema *sema, fn func(int)) acker {
	return buildClientCountACK(b.pipeline, canDrop, sema, func(guard *clientACKer) func(int, int) {
		return func(total, acked int) {
			if guard.Active() {
				fn(total)
			}
		}
	})
}

func (b *pipelineEmptyACK) createEventACKer(
	canDrop bool,
	sema *sema,
	fn func([]interface{}),
) acker {
	return buildClientEventACK(b.pipeline, canDrop, sema, func(guard *clientACKer) func([]interface{}, int) {
		return func(events []interface{}, acked int) {
			if guard.Active() {
				fn(events)
			}
		}
	})
}

type pipelineCountACK struct {
	pipeline *Pipeline
	cb       func(int, int)
}

func (b *pipelineCountACK) createPipelineACKer(canDrop bool, sema *sema) acker {
	return makeCountACK(b.pipeline, canDrop, sema, b.cb)
}

func (b *pipelineCountACK) createCountACKer(canDrop bool, sema *sema, fn func(int)) acker {
	return buildClientCountACK(b.pipeline, canDrop, sema, func(guard *clientACKer) func(int, int) {
		return func(total, acked int) {
			b.cb(total, acked)
			if guard.Active() {
				fn(total)
			}
		}
	})
}

func (b *pipelineCountACK) createEventACKer(
	canDrop bool,
	sema *sema,
	fn func([]interface{}),
) acker {
	return buildClientEventACK(b.pipeline, canDrop, sema, func(guard *clientACKer) func([]interface{}, int) {
		return func(data []interface{}, acked int) {
			b.cb(len(data), acked)
			if guard.Active() {
				fn(data)
			}
		}
	})
}

type pipelineEventsACK struct {
	pipeline *Pipeline
	cb       func([]interface{}, int)
}

func (b *pipelineEventsACK) createPipelineACKer(canDrop bool, sema *sema) acker {
	return newEventACK(b.pipeline, canDrop, sema, b.cb)
}

func (b *pipelineEventsACK) createCountACKer(canDrop bool, sema *sema, fn func(int)) acker {
	return buildClientEventACK(b.pipeline, canDrop, sema, func(guard *clientACKer) func([]interface{}, int) {
		return func(data []interface{}, acked int) {
			b.cb(data, acked)
			if guard.Active() {
				fn(len(data))
			}
		}
	})
}

func (b *pipelineEventsACK) createEventACKer(canDrop bool, sema *sema, fn func([]interface{})) acker {
	return buildClientEventACK(b.pipeline, canDrop, sema, func(guard *clientACKer) func([]interface{}, int) {
		return func(data []interface{}, acked int) {
			b.cb(data, acked)
			if guard.Active() {
				fn(data)
			}
		}
	})
}

// pipelineEventCB internally handles active ACKs in the pipeline.
// It receives ACK events from the queue and the individual clients.
// Once the queue returns an ACK to the pipelineEventCB, the worker loop will collect
// events from all clients having published events in the last batch of events
// being ACKed.
// the PipelineACKHandler will be notified, once all events being ACKed
// (including dropped events) have been collected. Only one ACK-event is handled
// at a time. The pipeline global and clients ACK handler will be blocked for the time
// an ACK event is being processed.
type pipelineEventCB struct {
	done chan struct{}

	acks chan int

	events        chan eventsDataMsg
	droppedEvents chan eventsDataMsg

	mode    pipelineACKMode
	handler beat.PipelineACKHandler
}

type eventsDataMsg struct {
	data         []interface{}
	total, acked int
	sig          chan struct{}
}

type pipelineACKMode uint8

const (
	noACKMode pipelineACKMode = iota
	countACKMode
	eventsACKMode
	lastEventsACKMode
)

func newPipelineEventCB(handler beat.PipelineACKHandler) (*pipelineEventCB, error) {
	mode := noACKMode
	if handler.ACKCount != nil {
		mode = countACKMode
	}
	if handler.ACKEvents != nil {
		if mode != noACKMode {
			return nil, errors.New("only one callback can be set")
		}
		mode = eventsACKMode
	}
	if handler.ACKLastEvents != nil {
		if mode != noACKMode {
			return nil, errors.New("only one callback can be set")
		}
		mode = lastEventsACKMode
	}

	// yay, no work
	if mode == noACKMode {
		return nil, nil
	}

	cb := &pipelineEventCB{
		acks:          make(chan int),
		mode:          mode,
		handler:       handler,
		events:        make(chan eventsDataMsg),
		droppedEvents: make(chan eventsDataMsg),
	}
	go cb.worker()
	return cb, nil
}

func (p *pipelineEventCB) close() {
	close(p.done)
}

// reportEvents sends a batch of ACKed events to the ACKer.
// The events array contains send and dropped events. The `acked` counters
// indicates the total number of events acked by the pipeline.
// That is, the number of dropped events is given by `len(events) - acked`.
// A client can report events with acked=0, iff the client has no waiting events
// in the pipeline (ACK ordering requirements)
//
// Note: the call blocks, until the ACK handler has collected all active events
//       from all clients. This ensure an ACK event being fully 'captured'
//       by the pipeline, before receiving/processing another ACK event.
//       In the meantime the queue has the chance of batching-up more ACK events,
//       such that only one ACK event is being reported to the pipeline handler
func (p *pipelineEventCB) onEvents(data []interface{}, acked int) {
	p.pushMsg(eventsDataMsg{data: data, total: len(data), acked: acked})
}

func (p *pipelineEventCB) onCounts(total, acked int) {
	p.pushMsg(eventsDataMsg{total: total, acked: acked})
}

func (p *pipelineEventCB) pushMsg(msg eventsDataMsg) {
	if msg.acked == 0 {
		p.droppedEvents <- msg
	} else {
		msg.sig = make(chan struct{})
		p.events <- msg
		<-msg.sig
	}
}

// Starts a new ACKed event.
func (p *pipelineEventCB) reportQueueACK(acked int) {
	p.acks <- acked
}

func (p *pipelineEventCB) worker() {
	defer close(p.acks)
	defer close(p.events)
	defer close(p.droppedEvents)

	for {
		select {
		case count := <-p.acks:
			exit := p.collect(count)
			if exit {
				return
			}

			// short circuite dropped events, but have client block until all events
			// have been processed by pipeline ack handler
		case msg := <-p.droppedEvents:
			p.reportEventsData(msg.data, msg.total)
			if msg.sig != nil {
				close(msg.sig)
			}

		case <-p.done:
			return
		}
	}
}

func (p *pipelineEventCB) collect(count int) (exit bool) {
	var (
		signalers []chan struct{}
		data      []interface{}
		acked     int
		total     int
	)

	for acked < count {
		var msg eventsDataMsg
		select {
		case msg = <-p.events:
		case msg = <-p.droppedEvents:
		case <-p.done:
			exit = true
			return
		}

		if msg.sig != nil {
			signalers = append(signalers, msg.sig)
		}
		total += msg.total
		acked += msg.acked

		if count-acked < 0 {
			panic("ack count mismatch")
		}

		switch p.mode {
		case eventsACKMode:
			data = append(data, msg.data...)

		case lastEventsACKMode:
			if L := len(msg.data); L > 0 {
				data = append(data, msg.data[L-1])
			}
		}
	}

	// signal clients we processed all active ACKs, as reported by queue
	for _, sig := range signalers {
		close(sig)
	}
	p.reportEventsData(data, total)
	return
}

func (p *pipelineEventCB) reportEventsData(data []interface{}, total int) {
	// report ACK back to the beat
	switch p.mode {
	case countACKMode:
		p.handler.ACKCount(total)
	case eventsACKMode:
		p.handler.ACKEvents(data)
	case lastEventsACKMode:
		p.handler.ACKLastEvents(data)
	}
}
