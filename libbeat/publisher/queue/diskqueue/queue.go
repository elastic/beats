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

package diskqueue

import (
	"fmt"
	"os"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// diskQueue is the internal type representing a disk-based implementation
// of queue.Queue.
type diskQueue struct {
	logger   *logp.Logger
	settings Settings

	// The persistent queue state (wraps diskQueuePersistentState on disk).
	stateFile *stateFile

	// Metadata related to the segment files.
	segments *diskQueueSegments

	// Metadata related to consumer acks / positions of the oldest remaining
	// frame.
	acks *diskQueueACKs

	// The queue's helper loops, each of which is run in its own goroutine.
	readerLoop  *readerLoop
	writerLoop  *writerLoop
	deleterLoop *deleterLoop

	// Wait group for shutdown of the goroutines associated with this queue:
	// reader loop, writer loop, deleter loop, and core loop (diskQueue.run()).
	waitGroup *sync.WaitGroup

	// The API channels used by diskQueueProducer to send write / cancel calls.
	producerWriteRequestChan  chan producerWriteRequest
	producerCancelRequestChan chan producerCancelRequest

	// When a consumer ack increments ackedUpTo, the consumer sends
	// its new value to this channel. The core loop then decides whether to
	// delete the containing segments.
	// The value sent on the channel is redundant with the value of ackedUpTo,
	// but we send it anyway so we don't have to worry about the core loop
	// waiting on ackLock.
	consumerACKChan chan frameID

	// writing is true if a writeRequest is currently being processed by the
	// writer loop, false otherwise.
	writing bool

	// reading is true if the reader loop is processing a readBlock, false
	// otherwise.
	reading bool

	// deleting is true if the segment-deletion loop is processing a deletion
	// request, false otherwise.
	deleting bool

	// pendingFrames is a list of all incoming data frames that have been
	// accepted by the queue and are waiting to be sent to the writer loop.
	// Segment ids in this list always appear in sorted order, even between
	// requests (that is, a frame added to this list always has segment id
	// at least as high as every previous frame that has ever been added).
	pendingFrames []segmentedFrame

	// blockedProducers is a list of all producer write requests that are
	// waiting for free space in the queue.
	blockedProducers []producerWriteRequest

	// This value represents the oldest frame ID for a segment that has not
	// yet been moved to the acked list. It is used to detect when the oldest
	// outstanding segment has been fully acknowledged by the consumer.
	oldestFrameID frameID

	// This lock must be held to read and write acked and ackedUpTo.
	ackLock sync.Mutex

	// The lowest frame id that has not yet been acknowledged.
	ackedUpTo frameID

	// A map of all acked indices that are above ackedUpTo (and thus
	// can't yet be acknowledged as a continuous block).
	// TODO: do this better.
	acked map[frameID]bool

	// The channel to signal our goroutines to shut down.
	done chan struct{}
}

// queuePosition represents a logical position within the queue buffer.
type queuePosition struct {
	segment segmentID
	offset  segmentOffset
}

type diskQueueACKs struct {
	// This lock must be held to access this structure.
	lock sync.Mutex

	// The id and position of the first unacknowledged frame.
	nextFrameID  frameID
	nextPosition queuePosition

	// A map of all acked indices that are above ackedUpTo (and thus
	// can't yet be acknowledged as a continuous block).
	// TODO: do this better.
	acked map[frameID]bool
}

func init() {
	queue.RegisterQueueType(
		"disk",
		queueFactory,
		feature.MakeDetails(
			"Disk queue",
			"Buffer events on disk before sending to the output.",
			feature.Beta))
}

// queueFactory matches the queue.Factory interface, and is used to add the
// disk queue to the registry.
func queueFactory(
	ackListener queue.ACKListener, logger *logp.Logger, cfg *common.Config,
) (queue.Queue, error) {
	settings, err := SettingsForUserConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("Disk queue couldn't load user config: %w", err)
	}
	settings.WriteToDiskListener = ackListener
	return NewQueue(logger, settings)
}

// NewQueue returns a disk-based queue configured with the given logger
// and settings, creating it if it doesn't exist.
func NewQueue(logger *logp.Logger, settings Settings) (queue.Queue, error) {
	logger = logger.Named("diskqueue")
	logger.Debugf(
		"Initializing disk queue at path %v", settings.directoryPath())

	if settings.MaxBufferSize > 0 &&
		settings.MaxBufferSize < settings.MaxSegmentSize*2 {
		return nil, fmt.Errorf(
			"Disk queue buffer size (%v) must be at least "+
				"twice the segment size (%v)",
			settings.MaxBufferSize, settings.MaxSegmentSize)
	}

	// Create the given directory path if it doesn't exist.
	err := os.MkdirAll(settings.directoryPath(), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("Couldn't create disk queue directory: %w", err)
	}

	// Index any existing data segments to be placed in segments.reading.
	initialSegments, err := scanExistingSegments(settings.directoryPath())
	if err != nil {
		return nil, err
	}
	var nextSegmentID segmentID
	if len(initialSegments) > 0 {
		lastID := initialSegments[len(initialSegments)-1].id
		nextSegmentID = lastID + 1
	}

	// Load the file handle for the queue state.
	stateFile, err := stateFileForPath(settings.stateFilePath())
	if err != nil {
		return nil, fmt.Errorf("Couldn't open disk queue metadata file: %w", err)
	}
	//if stateFile.loadedState.

	// We wait for four goroutines: core loop, reader loop, writer loop,
	// deleter loop.
	var waitGroup sync.WaitGroup
	waitGroup.Add(4)

	// The helper loops all have an input channel with buffer size 1, to ensure
	// that the core loop can never block when sending a request (the core
	// loop never sends a request until receiving the response from the
	// previous one, so there is never more than one outstanding request for
	// any helper loop).

	readerLoop := &readerLoop{
		settings: &settings,

		requestChan:  make(chan readerLoopRequest, 1),
		responseChan: make(chan readerLoopResponse),
		output:       make(chan *readFrame, settings.ReadAheadLimit),
		decoder:      newEventDecoder(),
	}
	go func() {
		readerLoop.run()
		waitGroup.Done()
	}()

	writerLoop := &writerLoop{
		logger:   logger,
		settings: &settings,

		requestChan:  make(chan writerLoopRequest, 1),
		responseChan: make(chan writerLoopResponse),
	}
	go func() {
		writerLoop.run()
		waitGroup.Done()
	}()

	deleterLoop := &deleterLoop{
		settings: &settings,

		requestChan:  make(chan deleterLoopRequest, 1),
		responseChan: make(chan deleterLoopResponse),
	}
	go func() {
		deleterLoop.run()
		waitGroup.Done()
	}()

	queue := &diskQueue{
		logger:   logger,
		settings: settings,

		stateFile: stateFile,
		segments: &diskQueueSegments{
			reading: initialSegments,
			nextID:  nextSegmentID,
		},

		readerLoop:  readerLoop,
		writerLoop:  writerLoop,
		deleterLoop: deleterLoop,

		// TODO: is this a reasonable size for this channel buffer?
		// Should this be customizable? (Tentatively: no, since we
		// expect most producer write requests to be handled near-
		// instantly so the requests are unlikely to accumulate).
		producerWriteRequestChan:  make(chan producerWriteRequest, 10),
		producerCancelRequestChan: make(chan producerCancelRequest),

		consumerACKChan: make(chan frameID),
		acked:           make(map[frameID]bool),

		waitGroup: &waitGroup,
		done:      make(chan struct{}),
	}

	// Start the queue's main loop.
	go func() {
		queue.run()
		waitGroup.Done()
	}()

	return queue, nil
}

//
// diskQueue implementation of the queue.Queue interface
//

func (dq *diskQueue) Close() error {
	// Closing the done channel signals to the core loop that it should
	// shut down the other helper goroutines and wrap everything up.
	close(dq.done)
	dq.waitGroup.Wait()

	return nil
}

func (dq *diskQueue) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{MaxEvents: 0}
}

func (dq *diskQueue) Producer(cfg queue.ProducerConfig) queue.Producer {
	return &diskQueueProducer{
		queue:   dq,
		config:  cfg,
		encoder: newEventEncoder(),
	}
}

func (dq *diskQueue) Consumer() queue.Consumer {
	return &diskQueueConsumer{
		queue:  dq,
		closed: false,
	}
}
