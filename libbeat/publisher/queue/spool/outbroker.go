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
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/go-txfile/pq"
)

type outBroker struct {
	ctx   *spoolCtx
	state func(*outBroker) bool

	// internal API
	sigFlushed chan uint
	get        chan getRequest

	// ack signaling
	pendingACKs   chanList      // list of pending batches to be forwarded to the ackLoop
	scheduledACKs chan chanList // shared channel for forwarding batches to ackLoop
	schedACKs     chan chanList // active ack forwarding channel, as used by broker (nil if pendingACKs is empty)

	// queue state
	queue     *pq.Queue
	reader    *pq.Reader
	available uint // number of available events. getRequests are only accepted if available > 0
	events    []publisher.Event
	required  int
	total     int
	active    getRequest

	// internal
	timer *timer
	dec   *decoder
}

type chanList struct {
	head *ackChan
	tail *ackChan
}

type ackChan struct {
	next  *ackChan
	ch    chan batchAckMsg
	total int // total number of events to ACK with this batch
}

const (
	// maximum number of events if getRequest size is <0
	maxEvents = 2048

	outSigChannelSize = 3
)

var ackChanPool = sync.Pool{
	New: func() interface{} {
		return &ackChan{
			ch: make(chan batchAckMsg, 1),
		}
	},
}

var errRetry = errors.New("retry")

func newOutBroker(ctx *spoolCtx, qu *pq.Queue, flushTimeout time.Duration) (*outBroker, error) {
	reader := qu.Reader()

	var (
		avail uint
		err   error
	)
	func() {
		if err = reader.Begin(); err != nil {
			return
		}
		defer reader.Done()
		avail, err = reader.Available()
	}()
	if err != nil {
		return nil, err
	}

	b := &outBroker{
		ctx:   ctx,
		state: nil,

		// API
		sigFlushed: make(chan uint, outSigChannelSize),
		get:        make(chan getRequest),

		// ack signaling
		pendingACKs:   chanList{},
		scheduledACKs: make(chan chanList),
		schedACKs:     nil,

		// queue state
		queue:     qu,
		reader:    reader,
		available: avail,
		events:    nil,
		required:  0,
		total:     0,
		active:    getRequest{},

		// internal
		timer: newTimer(flushTimeout),
		dec:   newDecoder(),
	}

	b.initState()
	ctx.Go(b.eventLoop)
	ctx.Go(b.ackLoop)
	return b, nil
}

func (b *outBroker) Consumer() *consumer {
	return newConsumer(b.ctx, b.get)
}

// onFlush is run whenever the queue flushes it's write buffer. The callback is
// run in the same go-routine as the Flush was executed from.
func (b *outBroker) onFlush(n uint) {
	if n > 0 {
		select {
		case <-b.ctx.Done(): // ignore flush messages on shutdown

		case b.sigFlushed <- n:

		}
	}
}

// onACK is run whenever the queue releases ACKed events. The number of acked
// events and freed pages will is reported.
func (b *outBroker) onACK(events, pages uint) {
}

func (b *outBroker) ackLoop() {
	log := b.ctx.logger

	log.Debug("start output ack loop")
	defer log.Debug("stop output ack loop")

	var ackList chanList // list of pending acks
	for {
		select {
		case <-b.ctx.Done():
			return

		case lst := <-b.scheduledACKs:
			ackList.concat(&lst)

		case <-ackList.channel():
			ackCh := ackList.pop()

			for {
				log.Debugf("receive ACK of %v events\n", ackCh.total)
				err := b.queue.ACK(uint(ackCh.total))
				if err != nil {
					log.Debugf("ack failed with: %+v", err)
					time.Sleep(1 * time.Second)
					continue
				}

				log.Debug("ACK succeeded")
				break
			}

			releaseACKChan(ackCh)
		}
	}
}

func (b *outBroker) eventLoop() {
	for {
		ok := b.state(b)
		if !ok {
			break
		}
	}
}

// initState resets the brokers state to the initial state and clears
// buffers/points from last state updates.
func (b *outBroker) initState() {
	b.events = nil
	b.required = 0
	b.total = 0
	b.active = getRequest{}
	if b.available == 0 {
		b.state = (*outBroker).stateWaitEvents
	} else {
		b.state = (*outBroker).stateActive
	}
}

// stateWaitEvents is the brokers state if the queue is empty.
// The broker waits for new events and does not accept and consumer requests.
//
// stateWaitEvents transitions:
//   -> stateActive: if a queue flush signal has been received
func (b *outBroker) stateWaitEvents() bool {
	log := b.ctx.logger
	log.Debug("outbroker (stateWaitEvents): waiting for new events")

	select {
	case <-b.ctx.Done():
		return false

	case n := <-b.sigFlushed:
		log.Debug("outbroker (stateWaitEvents): flush event", n)
		b.available += n
		b.state = (*outBroker).stateActive

	case b.schedACKs <- b.pendingACKs:
		b.handleACKsScheduled()
	}

	return true
}

// stateActive is the brokers initial state, waiting for consumer to request
// new events.
// Flush signals from the input are ignored.
//
// stateActive transitions:
//   -> stateActive: if consumer event get request has been fulfilled (N events
//                   copied or 0 timeout)
//   -> stateWaitEvents: if queue is empty after read
//   -> stateWithTimer: if only small number of events are available and flush
//                      timeout is configured.
func (b *outBroker) stateActive() bool {
	log := b.ctx.logger

	select {
	case <-b.ctx.Done():
		return false

	case n := <-b.sigFlushed:
		b.available += n

	case b.schedACKs <- b.pendingACKs:
		b.handleACKsScheduled()

	case req := <-b.get:
		var events []publisher.Event
		required := maxEvents
		if req.sz > 0 {
			events = make([]publisher.Event, 0, req.sz)
			required = req.sz
		}

		log.Debug("outbroker (stateActive): get request", required)

		var err error
		var total int
		events, total, err = b.collectEvents(events, required)
		required -= len(events)
		b.available -= uint(total)

		log.Debug("  outbroker (stateActive): events collected", len(events), total, err)

		// forward error to consumer and continue with current state
		if err != nil {
			log.Debug("  outbroker (stateActive): return error")
			b.returnError(req, events, total, err)
			b.initState()
			break
		}

		// enough events? Return
		if required == 0 || (len(events) > 0 && b.timer.Zero()) {
			log.Debug("  outbroker (stateActive): return events")
			b.returnEvents(req, events, total)
			b.initState() // prepare for next request
			break
		}

		// If no events have been decoded, signal an error to the consumer to retry.
		// Meanwhile reinitialize state, waiting for more events.
		if len(events) == 0 {
			b.returnError(req, nil, total, errRetry)
			b.initState()
			break
		}

		// not enough events -> start timer and try to collect more
		b.events = events
		b.required = required
		b.active = req
		b.total = total
		b.timer.Start()
		log.Debug("  outbroker (stateActive): switch to stateWithTimer")
		b.state = (*outBroker).stateWithTimer
	}

	return true
}

// stateWithTimer is the brokers active state, if the events read is less then
// the minimal number of requested events.
// Once the timer triggers or more events have been consumed, the get response
// will be send to the consumer.
//
// stateWithTimer transitions:
//   -> stateWithTimer: if some, but not enough events have been read from the
//                      queue
//   -> stateActive: if the timer triggers or enough events have been returned
//                   to the consumer
func (b *outBroker) stateWithTimer() bool {
	log := b.ctx.logger

	select {
	case <-b.ctx.Done():
		return false

	case b.schedACKs <- b.pendingACKs:
		b.handleACKsScheduled()

	case <-b.timer.C:
		b.timer.Stop(true)
		log.Debug("outbroker (stateWithTimer): flush timer")
		b.returnEvents(b.active, b.events, b.total)

		log.Debug("outbroker (stateWithTimer): switch to stateActive")
		b.initState()

	case n := <-b.sigFlushed:
		// yay, more events \o/

		b.available += n

		L := len(b.events)
		required := b.required
		events, total, err := b.collectEvents(b.events, required)
		b.available -= uint(total)
		collected := len(events) - L
		required -= collected
		total += b.total

		log.Debug("  outbroker (stateWithTimer): events collected", len(events), total, err)

		// continue with stateWithTimer?
		if err == nil && required > 0 {
			b.events = events
			b.total = total
			b.required = required
			log.Debug("  outbroker (stateWithTimer): switch to stateWithTimer")
			break
		}

		// done serving consumer request
		b.timer.Stop(false)
		if err != nil {
			log.Debug("  outbroker (stateWithTimer): return error")
			b.returnError(b.active, events, total, err)
		} else {
			log.Debug("  outbroker (stateWithTimer): return events")
			b.returnEvents(b.active, events, total)
		}

		log.Debug("outbroker (stateWithTimer): switch to stateActive")
		b.initState()
	}

	return true
}

func (b *outBroker) handleACKsScheduled() {
	b.schedACKs = nil
	b.pendingACKs = chanList{}
}

func (b *outBroker) newACKChan(total int) *ackChan {
	ackCh := newACKChan(total)
	b.pendingACKs.append(ackCh)
	b.schedACKs = b.scheduledACKs
	return ackCh
}

// signalDrop forwards an ACK of total events to the ackloop.
// The batch is marked as ACKed by the output.
// signalDrop is used to free space in the queue, in case
// a continuous set of events has been dropped due to decoding errors.
func (b *outBroker) signalDrop(total int) {
	ackCh := b.newACKChan(total)
	ackCh.ch <- batchAckMsg{}
}

func (b *outBroker) returnEvents(req getRequest, events []publisher.Event, total int) {
	ackCh := b.newACKChan(total)
	req.resp <- getResponse{
		ack: ackCh.ch,
		err: nil,
		buf: events,
	}
}

func (b *outBroker) returnError(
	req getRequest,
	events []publisher.Event,
	total int,
	err error,
) {
	var ch chan batchAckMsg

	if len(events) == 0 && total > 0 {
		b.signalDrop(total)
	}
	if len(events) > 0 {
		ackCh := b.newACKChan(total)
		ch = ackCh.ch
	}

	req.resp <- getResponse{
		ack: ch,
		err: err,
		buf: events,
	}
}

func (b *outBroker) collectEvents(
	events []publisher.Event,
	N int,
) ([]publisher.Event, int, error) {
	log := b.ctx.logger
	reader := b.reader

	// ensure all read operations happen within same transaction
	err := reader.Begin()
	if err != nil {
		return nil, 0, err
	}
	defer reader.Done()

	count := 0
	for N > 0 {
		sz, err := reader.Next()
		if sz <= 0 || err != nil {
			return events, count, err
		}

		count++

		buf := b.dec.Buffer(sz)
		_, err = reader.Read(buf)
		if err != nil {
			return events, count, err
		}

		event, err := b.dec.Decode()
		if err != nil {
			log.Debug("Failed to decode event from spool: %v", err)
			continue
		}

		events = append(events, event)
		N--
	}

	return events, count, nil
}

func newACKChan(total int) *ackChan {
	c := ackChanPool.Get().(*ackChan)
	c.next = nil
	c.total = total
	return c
}

func releaseACKChan(c *ackChan) {
	c.next = nil
	ackChanPool.Put(c)
}

func (l *chanList) append(ch *ackChan) {
	if l.head == nil {
		l.head = ch
	} else {
		l.tail.next = ch
	}
	l.tail = ch
}

func (l *chanList) concat(other *chanList) {
	if other.head == nil {
		return
	}

	if l.head == nil {
		*l = *other
		return
	}

	l.tail.next = other.head
	l.tail = other.tail
}

func (l *chanList) channel() chan batchAckMsg {
	if l.head == nil {
		return nil
	}
	return l.head.ch
}

func (l *chanList) pop() *ackChan {
	ch := l.head
	if ch != nil {
		l.head = ch.next
		if l.head == nil {
			l.tail = nil
		}
	}

	ch.next = nil
	return ch
}
