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

package memqueue

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

// Feature exposes a memory queue.
var Feature = queue.Feature("mem",
	create,
	feature.NewDetails(
		"Memory queue",
		"Buffer events in memory before sending to the output.",
		feature.Stable),
)

type Broker struct {
	done chan struct{}

	logger logger

	bufSize int
	// buf         brokerBuffer
	// minEvents   int
	// idleTimeout time.Duration

	// api channels
	events    chan pushRequest
	requests  chan getRequest
	pubCancel chan producerCancelRequest

	// internal channels
	acks          chan int
	scheduledACKs chan chanList

	eventer queue.Eventer

	// wait group for worker shutdown
	wg          sync.WaitGroup
	waitOnClose bool
}

type Settings struct {
	Eventer        queue.Eventer
	Events         int
	FlushMinEvents int
	FlushTimeout   time.Duration
	WaitOnClose    bool
}

type ackChan struct {
	next         *ackChan
	ch           chan batchAckMsg
	seq          uint
	start, count int // number of events waiting for ACK
	states       []clientState
}

type chanList struct {
	head *ackChan
	tail *ackChan
}

func init() {
	queue.RegisterType("mem", create)
}

func create(eventer queue.Eventer, logger *logp.Logger, cfg *common.Config) (queue.Queue, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = logp.L()
	}

	return NewBroker(logger, Settings{
		Eventer:        eventer,
		Events:         config.Events,
		FlushMinEvents: config.FlushMinEvents,
		FlushTimeout:   config.FlushTimeout,
	}), nil
}

// NewBroker creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewBroker(
	logger logger,
	settings Settings,
) *Broker {
	// define internal channel size for producer/client requests
	// to the broker
	chanSize := 20

	var (
		sz           = settings.Events
		minEvents    = settings.FlushMinEvents
		flushTimeout = settings.FlushTimeout
	)

	if minEvents < 1 {
		minEvents = 1
	}
	if minEvents > 1 && flushTimeout <= 0 {
		minEvents = 1
		flushTimeout = 0
	}
	if minEvents > sz {
		minEvents = sz
	}

	if logger == nil {
		logger = logp.NewLogger("memqueue")
	}

	b := &Broker{
		done:   make(chan struct{}),
		logger: logger,

		// broker API channels
		events:    make(chan pushRequest, chanSize),
		requests:  make(chan getRequest),
		pubCancel: make(chan producerCancelRequest, 5),

		// internal broker and ACK handler channels
		acks:          make(chan int),
		scheduledACKs: make(chan chanList),

		waitOnClose: settings.WaitOnClose,

		eventer: settings.Eventer,
	}

	var eventLoop interface {
		run()
		processACK(chanList, int)
	}

	if minEvents > 1 {
		eventLoop = newBufferingEventLoop(b, sz, minEvents, flushTimeout)
	} else {
		eventLoop = newDirectEventLoop(b, sz)
	}

	b.bufSize = sz
	ack := newACKLoop(b, eventLoop.processACK)

	b.wg.Add(2)
	go func() {
		defer b.wg.Done()
		eventLoop.run()
	}()
	go func() {
		defer b.wg.Done()
		ack.run()
	}()

	return b
}

func (b *Broker) Close() error {
	close(b.done)
	if b.waitOnClose {
		b.wg.Wait()
	}
	return nil
}

func (b *Broker) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{
		Events: b.bufSize,
	}
}

func (b *Broker) Producer(cfg queue.ProducerConfig) queue.Producer {
	return newProducer(b, cfg.ACK, cfg.OnDrop, cfg.DropOnCancel)
}

func (b *Broker) Consumer() queue.Consumer {
	return newConsumer(b)
}

var ackChanPool = sync.Pool{
	New: func() interface{} {
		return &ackChan{
			ch: make(chan batchAckMsg, 1),
		}
	},
}

func newACKChan(seq uint, start, count int, states []clientState) *ackChan {
	ch := ackChanPool.Get().(*ackChan)
	ch.next = nil
	ch.seq = seq
	ch.start = start
	ch.count = count
	ch.states = states
	return ch
}

func releaseACKChan(c *ackChan) {
	c.next = nil
	ackChanPool.Put(c)
}

func (l *chanList) prepend(ch *ackChan) {
	ch.next = l.head
	l.head = ch
	if l.tail == nil {
		l.tail = ch
	}
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

func (l *chanList) append(ch *ackChan) {
	if l.head == nil {
		l.head = ch
	} else {
		l.tail.next = ch
	}
	l.tail = ch
}

func (l *chanList) count() (elems, count int) {
	for ch := l.head; ch != nil; ch = ch.next {
		elems++
		count += ch.count
	}
	return
}

func (l *chanList) empty() bool {
	return l.head == nil
}

func (l *chanList) front() *ackChan {
	return l.head
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

func (l *chanList) reverse() {
	tmp := *l
	*l = chanList{}

	for !tmp.empty() {
		l.prepend(tmp.pop())
	}
}
