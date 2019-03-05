// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package spool

import (
	"fmt"
	"math"
	"time"

	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/go-txfile/pq"
)

type inBroker struct {
	ctx     *spoolCtx
	eventer queue.Eventer

	// active state handler
	state func(*inBroker) bool

	// api channels
	events    chan pushRequest
	pubCancel chan producerCancelRequest

	// queue signaling
	sigACK   chan struct{}
	sigFlush chan uint
	ackDone  chan struct{}

	// queue state
	queue        *pq.Queue
	writer       *pq.Writer
	clientStates clientStates

	// Event contents, that still needs to be send to the queue. An event is
	// pending if it has been serialized, but not added to the write buffer in
	// full, as some I/O operation on the write buffer failed.
	// =>
	//   - keep pointer to yet unwritten event contents
	//   - do not accept any events if pending is not nil
	//   - wait for signal from reader/queue-gc to retry writing the pending
	//     events contents
	pending []byte

	bufferedEvents uint // number of buffered events

	// flush settings
	timer       *timer
	flushEvents uint

	enc *encoder
}

const (
	inSigChannelSize   = 3
	inEventChannelSize = 20
)

func newInBroker(
	ctx *spoolCtx,
	eventer queue.Eventer,
	qu *pq.Queue,
	codec codecID,
	flushTimeout time.Duration,
	flushEvents uint,
) (*inBroker, error) {
	enc, err := newEncoder(codec)
	if err != nil {
		return nil, err
	}

	writer, err := qu.Writer()
	if err != nil {
		return nil, err
	}

	b := &inBroker{
		ctx:     ctx,
		eventer: eventer,
		state:   (*inBroker).stateEmpty,

		// API
		events:    make(chan pushRequest, inEventChannelSize),
		pubCancel: make(chan producerCancelRequest),
		sigACK:    make(chan struct{}, inSigChannelSize),
		sigFlush:  make(chan uint, inSigChannelSize),
		ackDone:   make(chan struct{}),

		// queue state
		queue:          qu,
		writer:         writer,
		clientStates:   clientStates{},
		pending:        nil,
		bufferedEvents: 0,

		// internal
		timer:       newTimer(flushTimeout),
		flushEvents: flushEvents,
		enc:         enc,
	}

	ctx.Go(b.eventLoop)
	ctx.Go(b.ackLoop)
	return b, nil
}

func (b *inBroker) Producer(cfg queue.ProducerConfig) queue.Producer {
	return newProducer(b.ctx, b.pubCancel, b.events, cfg.ACK, cfg.OnDrop, cfg.DropOnCancel)
}

// onFlush is run whenever the queue flushes it's write buffer. The callback is
// run in the same go-routine as the Flush was executed from.
// Only the (*inBroker).eventLoop triggers a flush.
func (b *inBroker) onFlush(n uint) {
	log := b.ctx.logger
	log.Debug("inbroker: onFlush ", n)

	if n == 0 {
		return
	}

	if b.eventer != nil {
		b.eventer.OnACK(int(n))
	}
	b.ctx.logger.Debug("inbroker: flushed events:", n)
	b.bufferedEvents -= n
	b.sigFlush <- n
}

// onACK is run whenever the queue releases ACKed events. The number of acked
// events and freed pages will is reported.
// Flush events are forward to the brokers eventloop, so to give the broker a
// chance to retry writing in case it has been blocked on a full queue.
func (b *inBroker) onACK(events, pages uint) {
	if pages > 0 {
		b.sigACK <- struct{}{}
	}
}

func (b *inBroker) ackLoop() {
	log := b.ctx.logger

	log.Debug("start flush ack loop")
	defer log.Debug("stop flush ack loop")

	for {
		var n uint
		select {
		case <-b.ackDone:
			return

		case n = <-b.sigFlush:
			log.Debug("inbroker: receive flush", n)
			states := b.clientStates.Pop(int(n))
			b.sendACKs(states)
		}
	}
}

// sendACKs returns the range of ACKed/Flushed events to the individual
// producers ACK handlers.
func (b *inBroker) sendACKs(states []clientState) {
	log := b.ctx.logger

	// reverse iteration on client states, so to report ranges of ACKed events
	// only once.
	N := len(states)
	total := 0
	for i := N - 1; i != -1; i-- {
		st := &states[i]
		if st.state == nil {
			continue
		}

		count := (st.seq - st.state.lastACK)
		if count == 0 || count > math.MaxUint32/2 {
			// seq number comparison did underflow. This happens only if st.seq has
			// already been acknowledged
			// log.Debug("seq number already acked: ", st.seq)

			st.state = nil
			continue
		}

		log.Debugf("broker ACK events: count=%v, start-seq=%v, end-seq=%v\n",
			count,
			st.state.lastACK+1,
			st.seq,
		)

		total += int(count)
		if total > N {
			panic(fmt.Sprintf("Too many events acked (expected=%v, total=%v)",
				N, total,
			))
		}

		// report range of ACKed events
		st.state.ackCB(int(count))
		st.state.lastACK = st.seq
		st.state = nil
	}
}

func (b *inBroker) eventLoop() {
	log := b.ctx.logger
	log.Info("spool input eventloop start")
	defer log.Info("spool input eventloop stop")

	// notify ackLoop to stop only after eventLoop has finished (after last flush)
	defer close(b.ackDone)
	defer b.eventloopShutdown()

	for {
		ok := b.state(b)
		if !ok {
			break
		}
	}
}

func (b *inBroker) eventloopShutdown() {
	// try to flush events/buffers on shutdown.
	if b.bufferedEvents == 0 {
		return
	}

	// Try to flush pending events.
	w := b.writer
	for len(b.pending) > 0 {
		n, err := w.Write(b.pending)
		b.pending = b.pending[n:]
		if err != nil {
			return
		}
	}
	w.Flush()
}

// stateEmpty is the brokers active state if the write buffer is empty and the
// queue did not block on write or flush operations.
// ACKs from the output are ignored, as events can still be added to the write
// buffer.
//
// stateEmpty transitions:
//   -> stateEmpty if serializing the event failed
//   -> stateWithTimer if event is written to buffer without flush
//        => start timer
//   -> stateBlocked if queue did return an error on write (Flush failed)
func (b *inBroker) stateEmpty() bool {
	log := b.ctx.logger

	select {
	case <-b.ctx.Done():
		return false

	case req := <-b.events:
		log.Debug("inbroker (stateEmpty): new event")

		buf, st, err := b.encodeEvent(&req)
		if err != nil {
			log.Debug("  inbroker (stateEmpty): encode failed")
			b.respondDrop(&req)
			break
		}

		// write/flush failed -> block until space in file becomes available
		err = b.addEvent(buf, st)
		if err != nil {
			log.Debug("  inbroker: append failed, blocking")
			b.state = (*inBroker).stateBlocked
			break
		}

		// start flush timer
		if b.flushEvents > 0 && b.bufferedEvents == b.flushEvents {
			log.Debug("  inbroker (stateEmpty): flush events")
			err := b.flushBuffer()
			if err != nil {
				log.Debug("  inbroker (stateEmpty): flush failed, blocking")
				b.state = (*inBroker).stateBlocked
			}
			break

		} else if b.bufferedEvents > 0 {
			log.Debug("  inbroker (stateEmpty): start flush timer")
			b.timer.Start()
			b.state = (*inBroker).stateWithTimer
		}

	case req := <-b.pubCancel:
		b.handleCancel(&req)

	case <-b.sigACK:
		// ignore ACKs as long as we can write without blocking
	}

	return true
}

// stateWithTimer is the brokers active state, if the write buffer is not empty.
// The flush timer is enabled as long as the broker is in this state.
// ACKs from the output are ignored, as events can still be added to the write
// buffer.
//
// stateWithTimer transitions:
//   -> stateWithTimer
//        - if serializing failed
//        - if event is added to buffer, without flush
//        - flush, but more events are available in the buffer (might reset timer)
//   -> stateEmpty if all events have been flushed
//   -> stateBlocked if queue did return an error on write/flush (Flush failed)
func (b *inBroker) stateWithTimer() bool {
	log := b.ctx.logger

	select {
	case <-b.ctx.Done():
		return false

	case req := <-b.events:
		log.Debug("inbroker (stateWithTimer): new event")

		buf, st, err := b.encodeEvent(&req)
		if err != nil {
			log.Debug("  inbroker (stateWithTimer): encode failed")
			b.respondDrop(&req)
			break
		}

		count := b.bufferedEvents
		err = b.addEvent(buf, st)
		if err != nil {
			log.Debug("  inbroker (stateWithTimer): append failed, blocking")
			b.state = (*inBroker).stateBlocked
			break
		}

		flushed := b.bufferedEvents < count
		if !flushed && b.flushEvents > 0 && b.bufferedEvents == b.flushEvents {
			err := b.flushBuffer()
			if err != nil {
				log.Debug("  inbroker (stateWithTimer): flush failed, blocking")
				b.state = (*inBroker).stateBlocked
				break
			}

			flushed = true
		}

		if !flushed {
			break
		}

		// write buffer has been flushed, reset timer and broker state
		log.Debug("  inbroker (stateWithTimer): buffer flushed")
		if b.bufferedEvents == 0 {
			b.timer.Stop(false)
			b.state = (*inBroker).stateEmpty
		} else {
			// restart timer, as new event is most likely the only event buffered
			// -> reduce IO
			log.Debug("  inbroker (stateWithTimer): start flush timer")
			b.timer.Restart()
		}

	case req := <-b.pubCancel:
		b.handleCancel(&req)

	case <-b.timer.C:
		log.Debug("inbroker (stateWithTimer): flush timeout", b.bufferedEvents)

		b.timer.Stop(true)

		err := b.flushBuffer()
		if err != nil {
			log.Debug("  inbroker (stateWithTimer): flush failed, blocking")
			b.state = (*inBroker).stateBlocked
			break
		}

		log.Debug("  inbroker (stateWithTimer): flush succeeded")

		if b.bufferedEvents > 0 {
			// flush did not push all events? Restart timer.
			log.Debug("  inbroker (stateWithTimer): start flush timer", b.bufferedEvents)
			b.timer.Start()
			break
		}

		b.state = (*inBroker).stateEmpty

	case <-b.sigACK:
		// ignore ACKs as long as we can write without blocking
	}

	return true
}

// stateBlocked is the brokers active state if the write buffer can not accept
// any new events.
// The broker will wait for an ACK signal from the outputs and retry flushing,
// in the hope of enough memory being available to flush the buffers.
// If flush did succeed, we try to add the pending event.
// For the time the broker is in this state, no events from any producers will
// be accepted. Thusly all producers will block. Closing a producer, unblocks
// the producer. The producers event (after close) might be processed or
// ignored in the future.
//
// stateBlocked transitions:
//   -> stateEmpty if flush was successful and write buffer is empty
//   -> stateWithTimer if flush was successful, but we still have some pending events
//   -> stateBlocked if flush failed (still not enough space)
func (b *inBroker) stateBlocked() bool {
	log := b.ctx.logger

	select {
	case <-b.ctx.Done():
		return false

	case req := <-b.pubCancel:
		b.handleCancel(&req)

	case <-b.sigACK:
		// TODO:
		//   Have write buffer report number of unallocated pages and take number
		//   of freed pages into account before retrying. This way no transaction
		//   must be created if it's already clear the flush will not succeed.

		log.Debug("inbroker (stateBlocked): ACK event from queue -> try to unblock")

		err := b.flushBuffer()
		if err != nil {
			log.Debug("  inbroker (stateBlocked): flush failed, blocking")
			break
		}

		if len(b.pending) > 0 {
			tmp := b.pending
			b.pending = nil
			err := b.writeEvent(tmp)
			if err != nil || len(b.pending) > 0 {
				log.Debugf("writing pending event failed: %+v", err)
				break
			}
		}

		if b.bufferedEvents == 0 {
			b.state = (*inBroker).stateEmpty
			break
		}

		b.timer.Start()
		log.Debug("  inbroker (stateBlocked): start flush timer")
		b.state = (*inBroker).stateWithTimer
	}

	return true
}

func (b *inBroker) handleCancel(req *producerCancelRequest) {
	// mark state as cancelled, so to not accept any new events
	// from the state object.
	if st := req.state; st != nil {
		st.cancelled = true
	}

	if req.resp != nil {
		req.resp <- producerCancelResponse{removed: 0}
	}
}

func (b *inBroker) encodeEvent(req *pushRequest) ([]byte, clientState, error) {
	buf, err := b.enc.encode(&req.event)
	if err != nil {
		return nil, clientState{}, err
	}

	if req.state == nil {
		return buf, clientState{}, nil
	}

	return buf, clientState{seq: req.seq, state: req.state}, nil
}

func (b *inBroker) respondDrop(req *pushRequest) {
	if req.state != nil {
		if cb := req.state.dropCB; cb != nil {
			cb(req.event.Content)
		}
	}
}

func (b *inBroker) addEvent(buf []byte, st clientState) error {
	log := b.ctx.logger

	b.bufferedEvents++
	log.Debug("  inbroker: add event of size", len(buf), b.bufferedEvents)

	count := b.clientStates.Add(st)
	log.Debug("  add event -> active:", count)

	err := b.writeEvent(buf)
	log.Debugf("  inbroker write -> events=%v, err=%+v ", b.bufferedEvents, err)

	return err
}

func (b *inBroker) writeEvent(buf []byte) error {
	log := b.ctx.logger

	// append event to queue
	w := b.writer
	n, err := w.Write(buf)
	buf = buf[n:]
	if len(buf) > 0 {
		b.pending = buf
	} else if err == nil {
		log.Debug("writer: finalize event in buffer")
		err = w.Next()
	}

	if err != nil {
		log.Debugf("Appending event content to write buffer failed with %+v", err)
	}
	return err
}

func (b *inBroker) flushBuffer() error {
	err := b.writer.Flush()
	if err != nil {
		log := b.ctx.logger
		log.Errorf("Spool flush failed with: %+v", err)
	}
	return err
}
