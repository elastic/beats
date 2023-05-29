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
	"io"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/opt"
)

// The string used to specify this queue in beats configurations.
const QueueType = "mem"

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

	// Consumers send requests to getChan to read events from the queue.
	getChan chan getRequest

	// Producers send requests to cancelChan to cancel events they've
	// sent so far that have not yet reached a consumer.
	cancelChan chan producerCancelRequest

	///////////////////////////
	// internal channels

	// When events are sent to consumers, the ACK channels for their batches
	// are collected into chanLists and sent to scheduledACKs.
	// These are then read by ackLoop and concatenated to its internal
	// chanList of all outstanding ACK channels.
	scheduledACKs chan chanList

	// A callback that should be invoked when ACKs are processed.
	// ackLoop calls this function when it advances the consumer ACK position.
	// Right now this forwards the notification to queueACKed() in
	// the pipeline observer, which updates the beats registry if needed.
	ackCallback func(eventCount int)

	// This channel is used to request/return metrics where such metrics require insight into
	// the actual eventloop itself. This seems like it might be overkill, but it seems that
	// all communication between the broker and the eventloops
	// happens via channels, so we're doing it this way.
	metricChan chan metricsRequest

	// wait group for worker shutdown
	wg sync.WaitGroup
}

type Settings struct {
	Events         int
	FlushMinEvents int
	FlushTimeout   time.Duration
	InputQueueSize int
}

type queueEntry struct {
	event interface{}
	id    queue.EntryID

	producer   *ackProducer
	producerID producerID // The order of this entry within its producer
}

type batch struct {
	queue    *broker
	entries  []queueEntry
	doneChan chan batchDoneMsg
}

// batchACKState stores the metadata associated with a batch of events sent to
// a consumer. When the consumer ACKs that batch, a batchAckMsg is sent on
// ackChan and received by
type batchACKState struct {
	next         *batchACKState
	doneChan     chan batchDoneMsg
	start, count int // number of events waiting for ACK
	entries      []queueEntry
}

type chanList struct {
	head *batchACKState
	tail *batchACKState
}

// FactoryForSettings is a simple wrapper around NewQueue so a concrete
// Settings object can be wrapped in a queue-agnostic interface for
// later use by the pipeline.
func FactoryForSettings(settings Settings) queue.QueueFactory {
	return func(
		logger *logp.Logger,
		ackCallback func(eventCount int),
	) (queue.Queue, error) {
		return NewQueue(logger, ackCallback, settings), nil
	}
}

// NewQueue creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewQueue(
	logger *logp.Logger,
	ackCallback func(eventCount int),
	settings Settings,
) *broker {
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
		scheduledACKs: make(chan chanList),

		ackCallback: ackCallback,
		metricChan:  make(chan metricsRequest),
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

func (b *broker) QueueType() string {
	return QueueType
}

func (b *broker) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{
		MaxEvents: b.bufSize,
	}
}

func (b *broker) Producer(cfg queue.ProducerConfig) queue.Producer {
	return newProducer(b, cfg.ACK, cfg.OnDrop, cfg.DropOnCancel)
}

func (b *broker) Get(count int) (queue.Batch, error) {
	responseChan := make(chan getResponse, 1)
	select {
	case <-b.done:
		return nil, io.EOF
	case b.getChan <- getRequest{
		entryCount: count, responseChan: responseChan}:
	}

	// if request has been sent, we have to wait for a response
	resp := <-responseChan
	return &batch{
		queue:    b,
		entries:  resp.entries,
		doneChan: resp.ackChan,
	}, nil
}

func (b *broker) Metrics() (queue.Metrics, error) {

	responseChan := make(chan memQueueMetrics, 1)
	select {
	case <-b.done:
		return queue.Metrics{}, io.EOF
	case b.metricChan <- metricsRequest{
		responseChan: responseChan}:
	}
	resp := <-responseChan

	return queue.Metrics{
		EventCount:            opt.UintWith(uint64(resp.currentQueueSize)),
		EventLimit:            opt.UintWith(uint64(b.bufSize)),
		UnackedConsumedEvents: opt.UintWith(uint64(resp.occupiedRead)),
		OldestEntryID:         resp.oldestEntryID,
	}, nil
}

var ackChanPool = sync.Pool{
	New: func() interface{} {
		return &batchACKState{
			doneChan: make(chan batchDoneMsg, 1),
		}
	},
}

func newBatchACKState(start, count int, entries []queueEntry) *batchACKState {
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

func (l *chanList) nextBatchChannel() chan batchDoneMsg {
	if l.head == nil {
		return nil
	}
	return l.head.doneChan
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

func (b *batch) Count() int {
	return len(b.entries)
}

func (b *batch) Entry(i int) interface{} {
	return b.entries[i].event
}

func (b *batch) FreeEntries() {
	// Memory queue can't release event references until they're fully acknowledged,
	// so do nothing.
}

func (b *batch) Done() {
	b.doneChan <- batchDoneMsg{}
}
