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

package proxyqueue

import (
	"io"
	"sync"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	c "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	minInputQueueSize      = 20
	maxInputQueueSizeRatio = 0.1
)

type broker struct {
	done chan struct{}

	logger *logp.Logger

	// The maximum number of events in any pending batch
	batchSize int

	///////////////////////////
	// api channels

	// Producers send requests to pushChan to add events to the queue.
	pushChan chan pushRequest

	// Consumers send requests to getChan to read events from the queue.
	getChan chan getRequest

	///////////////////////////
	// internal channels

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

	nextEntryID queue.EntryID
}

type Settings struct {
	ACKListener queue.ACKListener
	BatchSize   int
}

type queueEntry struct {
	event interface{}
	id    queue.EntryID

	producer *ackProducer
}

type ProxiedBatch struct {
	queue    *broker
	entries  []queueEntry
	doneChan chan batchDoneMsg
}

func (b *ProxiedBatch) FreeEntries() {
	b.entries = nil
}

// producerACKData tracks the number of events that need to be acknowledged
// from a single batch targeting a single producer.
type producerACKData struct {
	producer *ackProducer
	count    int
}

// batchACKState stores the metadata associated with a batch of events sent to
// a consumer. When the consumer ACKs that batch, its doneChan is closed.
// The run loop for the broker checks the doneChan for the next sequential
// outstanding batch (to ensure ACKs are delivered in order) and calls the
// producer's ackHandler when appropriate.
type batchACKState struct {
	next     *batchACKState
	doneChan chan batchDoneMsg
	acks     []producerACKData
}

type pendingACKsList struct {
	head *batchACKState
	tail *batchACKState
}

func (acks *pendingACKsList) nextDoneChan() chan batchDoneMsg {
	if acks.head != nil {
		return acks.head.doneChan
	}
	return nil
}

func init() {
	queue.RegisterQueueType(
		"proxy",
		create,
		feature.MakeDetails(
			"Proxy queue",
			"Pass through batched events to the Elastic Agent Shipper.",
			feature.Experimental))
}

func create(
	ackListener queue.ACKListener, logger *logp.Logger, cfg *c.C, inQueueSize int,
) (queue.Queue, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	if logger == nil {
		logger = logp.L()
	}

	return NewQueue(logger, Settings{
		ACKListener: ackListener,
		BatchSize:   50,
	}), nil
}

// NewQueue creates a new broker based in-memory queue holding up to sz number of events.
// If waitOnClose is set to true, the broker will block on Close, until all internal
// workers handling incoming messages and ACKs have been shut down.
func NewQueue(
	logger *logp.Logger,
	settings Settings,
) *broker {
	if logger == nil {
		logger = logp.NewLogger("proxyqueue")
	}

	b := &broker{
		done:   make(chan struct{}),
		logger: logger,

		// broker API channels
		pushChan: make(chan pushRequest, 8),
		getChan:  make(chan getRequest),

		ackListener: settings.ACKListener,
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.run()
	}()

	return b
}

func (b *broker) Close() error {
	close(b.done)
	return nil
}

func (b *broker) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{}
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
	return &ProxiedBatch{
		queue:    b,
		entries:  resp.entries,
		doneChan: resp.ackChan,
	}, nil
}

// Metrics returns an empty response because the proxy queue
// doesn't accumulate batches; for the real metadata, use either the
// Beats pipeline metrics, or the queue metrics in the shipper, which
// is where pending events are really queued when the proxy queue is
// in use.
func (b *broker) Metrics() (queue.Metrics, error) {
	return queue.Metrics{}, nil
}

var ackChanPool = sync.Pool{
	New: func() interface{} {
		return &batchACKState{
			doneChan: make(chan batchDoneMsg, 1),
		}
	},
}

func releaseACKChan(c *batchACKState) {
	c.next = nil
	ackChanPool.Put(c)
}

func (l *pendingACKsList) append(ch *batchACKState) {
	if l.head == nil {
		l.head = ch
	} else {
		l.tail.next = ch
	}
	l.tail = ch
}
func (l *pendingACKsList) nextBatchChannel() chan batchDoneMsg {
	if l.head == nil {
		return nil
	}
	return l.head.doneChan
}

func (l *pendingACKsList) pop() *batchACKState {
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

func (b *ProxiedBatch) Count() int {
	return len(b.entries)
}

func (b *ProxiedBatch) Entry(i int) interface{} {
	return b.entries[i].event
}

func (b *ProxiedBatch) ID(i int) queue.EntryID {
	return b.entries[i].id
}

func (b *ProxiedBatch) Done() {
	b.doneChan <- batchDoneMsg{}
}
