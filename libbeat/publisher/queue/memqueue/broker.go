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
type broker[T any] struct {
	settings Settings
	logger   *logp.Logger

	ctx       context.Context
	ctxCancel context.CancelFunc

	// The ring buffer backing the queue. All buffer positions should be taken
	// modulo the size of this array.
	buf []queueEntry[T]

	// wait group for queue workers (runLoop and ackLoop)
	wg sync.WaitGroup

	// The factory used to create an event encoder when creating a producer
	encoderFactory queue.EncoderFactory[T]

	///////////////////////////
	// api channels

	// Producers send requests to pushChan to add events to the queue.
	pushChan chan pushRequest[T]

	// Consumers send requests to getChan to read events from the queue.
	getChan chan getRequest[T]

	// Close triggers a queue close by sending to closeChan.
	// The value sent over this channel indicates if this is a force close.
	closeChan chan bool

	///////////////////////////
	// internal channels

	// Batches sent to consumers are also collected and forwarded to ackLoop
	// through this channel so ackLoop can monitor them for acknowledgments.
	consumedChan chan batchList[T]

	// When batches are acknowledged, ackLoop saves any metadata needed
	// for producer callbacks and such, then notifies runLoop that it's
	// safe to free these events and advance the queue by sending the
	// acknowledged event count to this channel.
	deleteChan chan int

	// closingChan is closed when the queue has processed a close request.
	// It's used to prevent producers from blocking on a closing queue.
	closingChan chan struct{}

	///////////////////////////////
	// internal goroutine state

	// The goroutine that manages the queue's core run state
	runLoop *runLoop[T]

	// The goroutine that manages ack notifications and callbacks
	ackLoop *ackLoop[T]

	///////////////////////////////
	// object caching
	batchPool sync.Pool
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

type queueEntry[T any] struct {
	event     T
	eventSize int
	id        queue.EntryID

	producer   *ackProducer[T]
	producerID producerID // The order of this entry within its producer
}

type batch[T any] struct {
	queue *broker[T]

	// Next batch in the containing batchList
	next *batch[T]

	// Position and length of the events within the queue buffer
	start, count int

	// batch.Done() sends to doneChan, where ackLoop reads it and handles
	// acknowledgment / cleanup.
	doneChan chan batchDoneMsg
}

type batchList[T any] struct {
	head *batch[T]
	tail *batch[T]
}

// FactoryForSettings is a simple wrapper around NewQueue so a concrete
// Settings object can be wrapped in a queue-agnostic interface for
// later use by the pipeline.
func FactoryForSettings[T any](settings Settings) queue.QueueFactory[T] {
	return func(
		logger *logp.Logger,
		observer queue.Observer,
		inputQueueSize int,
		encoderFactory queue.EncoderFactory[T],
	) (queue.Queue[T], error) {
		return NewQueue(logger, observer, settings, inputQueueSize, encoderFactory), nil
	}
}

// NewQueue creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewQueue[T any](
	logger *logp.Logger,
	observer queue.Observer,
	settings Settings,
	inputQueueSize int,
	encoderFactory queue.EncoderFactory[T],
) *broker[T] {
	b := newQueue(logger, observer, settings, inputQueueSize, encoderFactory)

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
func newQueue[T any](
	logger *logp.Logger,
	observer queue.Observer,
	settings Settings,
	inputQueueSize int,
	encoderFactory queue.EncoderFactory[T],
) *broker[T] {
	if observer == nil {
		observer = queue.NewQueueObserver(nil)
	}
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
	} else {
		logger = logger.Named("memqueue")
	}

	b := &broker[T]{
		settings: settings,
		logger:   logger,

		buf: make([]queueEntry[T], settings.Events),

		encoderFactory: encoderFactory,

		// broker API channels
		pushChan:  make(chan pushRequest[T], chanSize),
		getChan:   make(chan getRequest[T]),
		closeChan: make(chan bool),

		// internal runLoop and ackLoop channels
		consumedChan: make(chan batchList[T]),
		deleteChan:   make(chan int),
		closingChan:  make(chan struct{}),

		// reuse pool for batch objects
		batchPool: sync.Pool{
			New: func() interface{} {
				return &batch[T]{
					doneChan: make(chan batchDoneMsg, 1),
				}
			},
		},
	}
	b.ctx, b.ctxCancel = context.WithCancel(context.Background())

	b.runLoop = newRunLoop(b, observer)
	b.ackLoop = newACKLoop(b)

	observer.MaxEvents(settings.Events)

	return b
}

func (b *broker[T]) Close(force bool) error {
	select {
	case b.closeChan <- force:
	case <-b.ctx.Done():
	}

	return nil
}

func (b *broker[T]) Done() <-chan struct{} {
	return b.ctx.Done()
}

func (b *broker[T]) QueueType() string {
	return QueueType
}

func (b *broker[T]) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{
		MaxEvents: len(b.buf),
	}
}

func (b *broker[T]) Producer(cfg queue.ProducerConfig) queue.Producer[T] {
	// If we were given an encoder factory to allow producers to encode
	// events for output before they entered the queue, then create an
	// encoder for the new producer.
	var encoder queue.Encoder[T]
	if b.encoderFactory != nil {
		encoder = b.encoderFactory()
	}
	return newProducer(b, cfg.ACK, encoder)
}

func (b *broker[T]) Get(count int) (queue.Batch[T], error) {
	responseChan := make(chan *batch[T], 1)
	select {
	case <-b.ctx.Done():
		return nil, io.EOF
	case b.getChan <- getRequest[T]{
		entryCount: count, responseChan: responseChan}:
	}

	// if request has been sent, we have to wait for a response
	resp := <-responseChan
	return resp, nil
}

func newBatch[T any](queue *broker[T], start, count int) *batch[T] {
	batch := queue.batchPool.Get().(*batch[T]) //nolint:errcheck //safe to ignore type check
	batch.next = nil
	batch.queue = queue
	batch.start = start
	batch.count = count
	return batch
}

func releaseBatch[T any](b *batch[T]) {
	b.next = nil
	b.queue.batchPool.Put(b)
}

func (l *batchList[T]) prepend(b *batch[T]) {
	b.next = l.head
	l.head = b
	if l.tail == nil {
		l.tail = b
	}
}

func (l *batchList[T]) concat(other *batchList[T]) {
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

func (l *batchList[T]) append(b *batch[T]) {
	if l.head == nil {
		l.head = b
	} else {
		l.tail.next = b
	}
	l.tail = b
}

func (l *batchList[T]) empty() bool {
	return l.head == nil
}

func (l *batchList[T]) front() *batch[T] {
	return l.head
}

func (l *batchList[T]) nextBatchChannel() chan batchDoneMsg {
	if l.head == nil {
		return nil
	}
	return l.head.doneChan
}

func (l *batchList[T]) pop() *batch[T] {
	ch := l.head
	if ch != nil {
		l.head = ch.next
		if l.head == nil {
			l.tail = nil
		}
		ch.next = nil
	}

	return ch
}

func (l *batchList[T]) reverse() {
	tmp := *l
	*l = batchList[T]{}

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

func (b *batch[T]) Count() int {
	return b.count
}

// Return a pointer to the queueEntry for the i-th element of this batch
func (b *batch[T]) rawEntry(i int) *queueEntry[T] {
	// Indexes wrap around the end of the queue buffer
	return &b.queue.buf[(b.start+i)%len(b.queue.buf)]
}

// Return the event referenced by the i-th element of this batch
func (b *batch[T]) Entry(i int) T {
	return b.rawEntry(i).event
}

func (b *batch[T]) FreeEntries() {
	// This signals that the event data has been copied out of the batch, and is
	// safe to free from the queue buffer, so set all the event pointers to nil.
	var empty T
	for i := 0; i < b.count; i++ {
		index := (b.start + i) % len(b.queue.buf)
		b.queue.buf[index].event = empty
	}
}

func (b *batch[T]) Done() {
	b.doneChan <- batchDoneMsg{}
}
