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
	"context"
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

// broker is the main implementation type for the memory queue. An active queue
// consists of two goroutines: runLoop, which handles all public API requests
// and owns the buffer state, and ackLoop, which listens for acknowledgments of
// consumed events and runs any appropriate completion handlers.
type broker struct {
	settings Settings
	logger   *logp.Logger

	ctx       context.Context
	ctxCancel context.CancelFunc

	// The ring buffer backing the queue. All buffer positions should be taken
	// modulo the size of this array.
	buf []queueEntry

	// wait group for queue workers (runLoop and ackLoop)
	wg sync.WaitGroup

	// The factory used to create an event encoder when creating a producer
	encoderFactory queue.EncoderFactory

	///////////////////////////
	// api channels

	// Producers send requests to pushChan to add events to the queue.
	pushChan chan pushRequest

	// Consumers send requests to getChan to read events from the queue.
	getChan chan getRequest

	// Producers send requests to cancelChan to cancel events they've
	// sent so far that have not yet reached a consumer.
	cancelChan chan producerCancelRequest

	// Metrics() sends requests to metricChan to expose internal queue
	// metrics to external callers.
	metricChan chan metricsRequest

	///////////////////////////
	// internal channels

	// Batches sent to consumers are also collected and forwarded to ackLoop
	// through this channel so ackLoop can monitor them for acknowledgments.
	consumedChan chan batchList

	// ackCallback is a configurable callback to invoke when ACKs are processed.
	// ackLoop calls this function when it advances the consumer ACK position.
	// Right now this forwards the notification to queueACKed() in
	// the pipeline observer, which updates the beats registry if needed.
	ackCallback func(eventCount int)

	// When batches are acknowledged, ackLoop saves any metadata needed
	// for producer callbacks and such, then notifies runLoop that it's
	// safe to free these events and advance the queue by sending the
	// acknowledged event count to this channel.
	deleteChan chan int

	///////////////////////////////
	// internal goroutine state

	// The goroutine that manages the queue's core run state
	runLoop *runLoop

	// The goroutine that manages ack notifications and callbacks
	ackLoop *ackLoop
}

type Settings struct {
	// The number of events the queue can hold.
	Events int

	// The most events that will ever be returned from one Get request.
	MaxGetRequest int

	// If positive, the amount of time the queue will wait to fill up
	// a batch if a Get request asks for more events than we have.
	FlushTimeout time.Duration
}

type queueEntry struct {
	event queue.Entry
	id    queue.EntryID

	producer   *ackProducer
	producerID producerID // The order of this entry within its producer
}

type batch struct {
	queue *broker

	// Next batch in the containing batchList
	next *batch

	// Position and length of the events within the queue buffer
	start, count int

	// batch.Done() sends to doneChan, where ackLoop reads it and handles
	// acknowledgment / cleanup.
	doneChan chan batchDoneMsg
}

type batchList struct {
	head *batch
	tail *batch
}

// FactoryForSettings is a simple wrapper around NewQueue so a concrete
// Settings object can be wrapped in a queue-agnostic interface for
// later use by the pipeline.
func FactoryForSettings(settings Settings) queue.QueueFactory {
	return func(
		logger *logp.Logger,
		ackCallback func(eventCount int),
		inputQueueSize int,
		encoderFactory queue.EncoderFactory,
	) (queue.Queue, error) {
		return NewQueue(logger, ackCallback, settings, inputQueueSize, encoderFactory), nil
	}
}

// NewQueue creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewQueue(
	logger *logp.Logger,
	ackCallback func(eventCount int),
	settings Settings,
	inputQueueSize int,
	encoderFactory queue.EncoderFactory,
) *broker {
	b := newQueue(logger, ackCallback, settings, inputQueueSize, encoderFactory)

	// Start the queue workers
	b.wg.Add(2)
	go func() {
		defer b.wg.Done()
		b.runLoop.run()
	}()
	go func() {
		defer b.wg.Done()
		b.ackLoop.run()
	}()

	return b
}

// newQueue does most of the work of creating a queue from the given
// parameters, but doesn't start the runLoop or ackLoop workers. This
// lets us perform more granular / deterministic tests by controlling
// when the workers are active.
func newQueue(
	logger *logp.Logger,
	ackCallback func(eventCount int),
	settings Settings,
	inputQueueSize int,
	encoderFactory queue.EncoderFactory,
) *broker {
	chanSize := AdjustInputQueueSize(inputQueueSize, settings.Events)

	// Backwards compatibility: an old way to select synchronous queue
	// behavior was to set "flush.min_events" to 0 or 1, in which case the
	// timeout was disabled and the max get request was half the queue.
	// (Otherwise, it would make sense to leave FlushTimeout unchanged here.)
	if settings.MaxGetRequest <= 1 {
		settings.FlushTimeout = 0
		settings.MaxGetRequest = (settings.Events + 1) / 2
	}

	// Can't request more than the full queue
	if settings.MaxGetRequest > settings.Events {
		settings.MaxGetRequest = settings.Events
	}

	if logger == nil {
		logger = logp.NewLogger("memqueue")
	}

	b := &broker{
		settings: settings,
		logger:   logger,

		buf: make([]queueEntry, settings.Events),

		encoderFactory: encoderFactory,

		// broker API channels
		pushChan:   make(chan pushRequest, chanSize),
		getChan:    make(chan getRequest),
		cancelChan: make(chan producerCancelRequest, 5),
		metricChan: make(chan metricsRequest),

		// internal runLoop and ackLoop channels
		consumedChan: make(chan batchList),
		deleteChan:   make(chan int),

		ackCallback: ackCallback,
	}
	b.ctx, b.ctxCancel = context.WithCancel(context.Background())

	b.runLoop = newRunLoop(b)
	b.ackLoop = newACKLoop(b)

	return b
}

func (b *broker) Close() error {
	b.ctxCancel()
	return nil
}

func (b *broker) QueueType() string {
	return QueueType
}

func (b *broker) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{
		MaxEvents: len(b.buf),
	}
}

func (b *broker) Producer(cfg queue.ProducerConfig) queue.Producer {
	// If we were given an encoder factory to allow producers to encode
	// events for output before they entered the queue, then create an
	// encoder for the new producer.
	var encoder queue.Encoder
	if b.encoderFactory != nil {
		encoder = b.encoderFactory()
	}
	return newProducer(b, cfg.ACK, cfg.OnDrop, cfg.DropOnCancel, encoder)
}

func (b *broker) Get(count int) (queue.Batch, error) {
	responseChan := make(chan *batch, 1)
	select {
	case <-b.ctx.Done():
		return nil, io.EOF
	case b.getChan <- getRequest{
		entryCount: count, responseChan: responseChan}:
	}

	// if request has been sent, we have to wait for a response
	resp := <-responseChan
	return resp, nil
}

func (b *broker) Metrics() (queue.Metrics, error) {

	responseChan := make(chan memQueueMetrics, 1)
	select {
	case <-b.ctx.Done():
		return queue.Metrics{}, io.EOF
	case b.metricChan <- metricsRequest{
		responseChan: responseChan}:
	}
	resp := <-responseChan

	return queue.Metrics{
		EventCount:            opt.UintWith(uint64(resp.currentQueueSize)),
		EventLimit:            opt.UintWith(uint64(len(b.buf))),
		UnackedConsumedEvents: opt.UintWith(uint64(resp.occupiedRead)),
		OldestEntryID:         resp.oldestEntryID,
	}, nil
}

var batchPool = sync.Pool{
	New: func() interface{} {
		return &batch{
			doneChan: make(chan batchDoneMsg, 1),
		}
	},
}

func newBatch(queue *broker, start, count int) *batch {
	batch := batchPool.Get().(*batch)
	batch.next = nil
	batch.queue = queue
	batch.start = start
	batch.count = count
	return batch
}

func releaseBatch(b *batch) {
	b.next = nil
	batchPool.Put(b)
}

func (l *batchList) prepend(b *batch) {
	b.next = l.head
	l.head = b
	if l.tail == nil {
		l.tail = b
	}
}

func (l *batchList) concat(other *batchList) {
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

func (l *batchList) append(b *batch) {
	if l.head == nil {
		l.head = b
	} else {
		l.tail.next = b
	}
	l.tail = b
}

func (l *batchList) empty() bool {
	return l.head == nil
}

func (l *batchList) front() *batch {
	return l.head
}

func (l *batchList) nextBatchChannel() chan batchDoneMsg {
	if l.head == nil {
		return nil
	}
	return l.head.doneChan
}

func (l *batchList) pop() *batch {
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

func (l *batchList) reverse() {
	tmp := *l
	*l = batchList{}

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
	return b.count
}

// Return a pointer to the queueEntry for the i-th element of this batch
func (b *batch) rawEntry(i int) *queueEntry {
	// Indexes wrap around the end of the queue buffer
	return &b.queue.buf[(b.start+i)%len(b.queue.buf)]
}

// Return the event referenced by the i-th element of this batch
func (b *batch) Entry(i int) queue.Entry {
	return b.rawEntry(i).event
}

func (b *batch) FreeEntries() {
	// This signals that the event data has been copied out of the batch, and is
	// safe to free from the queue buffer, so set all the event pointers to nil.
	for i := 0; i < b.count; i++ {
		index := (b.start + i) % len(b.queue.buf)
		b.queue.buf[index].event = nil
	}
}

func (b *batch) Done() {
	b.doneChan <- batchDoneMsg{}
}
