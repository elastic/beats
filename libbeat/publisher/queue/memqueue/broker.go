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

	///////////////////////////
	// api channels

	// Producers send requests to pushChan to add events to the queue.
	pushChan chan pushRequest

	// Consumers send getChan to getChan to read events from the queue.
	getChan chan getRequest

	// Producers send requests to cancelChan to cancel events they've
	// sent so far that have not yet reached a consumer.
	cancelChan chan producerCancelRequest

	///////////////////////////
	// internal channels

	// When ackLoop receives events ACKs from a consumer, it sends the number
	// of ACKed events to ackChan to notify the event loop that those
	// events can be removed from the queue.
	ackChan chan int

	// When events are sent to consumers, the ACK channels for their batches
	// are collected into chanLists and sent to scheduledACKs.
	// These are then read by ackLoop and concatenated to its internal
	// chanList of all outstanding ACK channels.
	scheduledACKs chan chanList

	// A listener that should be notified when ACKs are processed.
	// ackLoop calls this listener's OnACK function when it advances
	// the consumer ACK position.
	// Right now this listener always points at the Pipeline associated with
	// this queue. Pipeline.OnACK then forwards the notification to
	// Pipeline.observer.queueACKed(), which updates the beats registry
	// if needed.
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

// batchACKState stores the metadata associated with a batch of events sent to
// a consumer. When the consumer ACKs that batch, a batchAckMsg is sent on
// ackChan and received by
type batchACKState struct {
	next         *batchACKState
	ackChan      chan batchAckMsg
	start, count int // number of events waiting for ACK
	entries      []queueEntry
}

type chanList struct {
	head *batchACKState
	tail *batchACKState
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
		pushChan:   make(chan pushRequest, chanSize),
		getChan:    make(chan getRequest),
		cancelChan: make(chan producerCancelRequest, 5),

		// internal broker and ACK handler channels
		ackChan:       make(chan int),
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
	ackLoop := &ackLoop{
		broker:     b,
		processACK: eventLoop.processACK}

	b.wg.Add(2)
	go func() {
		defer b.wg.Done()
		eventLoop.run()
	}()
	go func() {
		defer b.wg.Done()
		ackLoop.run()
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
		return &batchACKState{
			ackChan: make(chan batchAckMsg, 1),
		}
	},
}

func newBatchACKState(start, count int, entries []queueEntry) *batchACKState {
	//nolint: errcheck // Return value doesn't need to be checked before conversion.
	ch := ackChanPool.Get().(*batchACKState)
	ch.next = nil
	ch.start = start
	ch.count = count
	ch.entries = entries
	return ch
}

func releaseACKChan(c *batchACKState) {
	c.next = nil
	ackChanPool.Put(c)
}

func (l *chanList) prepend(ch *batchACKState) {
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

func (l *chanList) append(ch *batchACKState) {
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

func (l *chanList) front() *batchACKState {
	return l.head
}

func (l *chanList) channel() chan batchAckMsg {
	if l.head == nil {
		return nil
	}
	return l.head.ackChan
}

func (l *chanList) pop() *batchACKState {
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
