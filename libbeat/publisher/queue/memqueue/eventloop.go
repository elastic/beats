package memqueue

import (
	"time"
)

type eventLoop struct {
	broker *Broker

	// active broker API channels
	events    chan pushRequest
	get       chan getRequest
	pubCancel chan producerCancelRequest

	// ack handling
	acks        chan int      // ackloop -> eventloop : total number of events ACKed by outputs
	schedACKS   chan chanList // eventloop -> ackloop : active list of batches to be acked
	pendingACKs chanList      // ordered list of active batches to be send to the ackloop
	ackSeq      uint          // ack batch sequence number to validate ordering

	// buffer flush timer state
	timer      *time.Timer
	idleC      <-chan time.Time
	forceFlush bool
}

func newEventLoop(b *Broker) *eventLoop {
	l := &eventLoop{
		broker:    b,
		events:    b.events,
		pubCancel: b.pubCancel,
		acks:      b.acks,
	}

	if to := b.idleTimeout; to > 0 {
		// create initialy 'stopped' timer -> reset will be used
		// on timer object, if flush timer becomes active.
		l.timer = time.NewTimer(to)
		if !l.timer.Stop() {
			<-l.timer.C
		}
	}

	return l
}

func (l *eventLoop) run() {
	broker := l.broker
	for {
		select {
		case <-broker.done:
			return

		case req := <-l.events: // producer pushing new event
			l.handleInsert(&req)

		case req := <-l.pubCancel: // producer cancellig active events
			l.handleCancel(&req)

		case req := <-l.get: // consumer asking for next batch
			l.handleConsumer(&req)

		case <-l.idleC:
			// handle flush timer being triggered -> pending events can be forwarded via 'get'
			l.enableFlushEvents()

		case l.schedACKS <- l.pendingACKs:
			// on send complete list of pending batches has been forwarded -> clear list and queue
			l.schedACKS = nil
			l.pendingACKs = chanList{}

		case count := <-l.acks:
			l.handleACK(count)

		}

		// update get and idle timer after state machine
		l.get = broker.requests
		if !l.forceFlush {
			avail := broker.avail()
			if avail == 0 || broker.totalAvail() < broker.minEvents {
				l.get = nil

				if avail > 0 {
					l.startFlushTimer()
				}
			}
		}
	}
}

func (l *eventLoop) handleInsert(req *pushRequest) {
	// log := l.broker.logger
	// log.Debugf("push event: %v\t%v\t%p\n", req.event, req.seq, req.state)

	if avail, ok := l.broker.insert(req); ok && avail == 0 {
		// log.Debugf("buffer: all regions full")

		// no more space to accept new events -> unset events queue for time being
		l.events = nil
	}
}

func (l *eventLoop) handleCancel(req *producerCancelRequest) {
	// log := l.broker.logger
	// log.Debug("handle cancel request")

	var (
		removed int
		broker  = l.broker
	)

	if st := req.state; st != nil {
		st.cancelled = true
		removed = broker.cancel(st)
	}

	// signal cancel request being finished
	if req.resp != nil {
		req.resp <- producerCancelResponse{
			removed: removed,
		}
	}

	// re-enable pushRequest if buffer can take new events
	if !broker.full() {
		l.events = broker.events
	}
}

func (l *eventLoop) handleConsumer(req *getRequest) {
	// log := l.broker.logger

	start, buf := l.broker.get(req.sz)
	count := len(buf)
	if count == 0 {
		panic("empty batch returned")
	}

	// log.Debug("newACKChan: ", b.ackSeq, count)
	ackCH := newACKChan(l.ackSeq, start, count)
	l.ackSeq++

	req.resp <- getResponse{buf, ackCH}
	l.pendingACKs.append(ackCH)
	l.schedACKS = l.broker.scheduledACKs

	l.stopFlushTimer()
}

func (l *eventLoop) handleACK(count int) {
	// log := l.broker.logger
	// log.Debug("receive buffer ack:", count)

	// Give broker/buffer a chance to clean up most recent ACKs
	// After handling ACKs some buffer has been freed up
	// -> always reenable producers
	broker := l.broker
	broker.cleanACKs(count)
	l.events = l.broker.events
}

func (l *eventLoop) enableFlushEvents() {
	l.forceFlush = true
	l.idleC = nil
}

func (l *eventLoop) stopFlushTimer() {
	l.forceFlush = false
	if l.idleC != nil {
		l.idleC = nil
		if !l.timer.Stop() {
			<-l.timer.C
		}
	}
}

func (l *eventLoop) startFlushTimer() {
	if l.idleC == nil && l.timer != nil {
		l.timer.Reset(l.broker.idleTimeout)
		l.idleC = l.timer.C
	}
}
