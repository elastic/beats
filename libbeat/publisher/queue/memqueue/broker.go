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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

const (
	minInputQueueSize      = 20
	maxInputQueueSizeRatio = 0.1
)

type broker struct {
	done chan struct{}

	logger *logp.Logger

	bufSize int

	// api channels
	events    chan pushRequest
	requests  chan getRequest
	pubCancel chan producerCancelRequest

	// internal channels
	acks          chan int
	scheduledACKs chan chanList

	ackListener queue.ACKListener

	// wait group for worker shutdown
	wg sync.WaitGroup
}

type Settings struct {
	ACKListener    queue.ACKListener
	Events         int
	FlushMinEvents int
	FlushTimeout   time.Duration
	InputQueueSize int
}

type ackChan struct {
	next         *ackChan
	ch           chan batchAckMsg
	seq          uint
	start, count int // number of events waiting for ACK
	entries      []queueEntry
}

type chanList struct {
	head *ackChan
	tail *ackChan
}

func init() {
	queue.RegisterQueueType(
		"mem",
		create,
		feature.MakeDetails(
			"Memory queue",
			"Buffer events in memory before sending to the output.",
			feature.Stable))
}

func create(
	ackListener queue.ACKListener, logger *logp.Logger, cfg *common.Config, inQueueSize int,
) (queue.Queue, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = logp.L()
	}

	return NewQueue(logger, Settings{
		ACKListener:    ackListener,
		Events:         config.Events,
		FlushMinEvents: config.FlushMinEvents,
		FlushTimeout:   config.FlushTimeout,
		InputQueueSize: inQueueSize,
	}), nil
}

// NewQueue creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewQueue(
	logger *logp.Logger,
	settings Settings,
) queue.Queue {
	var (
		sz           = settings.Events
		minEvents    = settings.FlushMinEvents
		flushTimeout = settings.FlushTimeout
	)

	chanSize := AdjustInputQueueSize(settings.InputQueueSize, sz)

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

	b := &broker{
		done:   make(chan struct{}),
		logger: logger,

		// broker API channels
		events:    make(chan pushRequest, chanSize),
		requests:  make(chan getRequest),
		pubCancel: make(chan producerCancelRequest, 5),

		// internal broker and ACK handler channels
		acks:          make(chan int),
		scheduledACKs: make(chan chanList),

		ackListener: settings.ACKListener,
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
	ack := &ackLoop{
		broker:     b,
		processACK: eventLoop.processACK}

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

func (b *broker) Close() error {
	close(b.done)
	return nil
}

func (b *broker) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{
		MaxEvents: b.bufSize,
	}
}

func (b *broker) Producer(cfg queue.ProducerConfig) queue.Producer {
	return newProducer(b, cfg.ACK, cfg.OnDrop, cfg.DropOnCancel)
}

func (b *broker) Consumer() queue.Consumer {
	return newConsumer(b)
}

var ackChanPool = sync.Pool{
	New: func() interface{} {
		return &ackChan{
			ch: make(chan batchAckMsg, 1),
		}
	},
}

func newACKChan(seq uint, start, count int, entries []queueEntry) *ackChan {
	//nolint: errcheck // Return value doesn't need to be checked before conversion.
	ch := ackChanPool.Get().(*ackChan)
	ch.next = nil
	ch.seq = seq
	ch.start = start
	ch.count = count
	ch.entries = entries
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

// AdjustInputQueueSize decides the size for the input queue.
func AdjustInputQueueSize(requested, mainQueueSize int) (actual int) {
	actual = requested
	if max := int(float64(mainQueueSize) * maxInputQueueSizeRatio); mainQueueSize > 0 && actual > max {
		actual = max
	}
	if actual < minInputQueueSize {
		actual = minInputQueueSize
	}
	return actual
}
