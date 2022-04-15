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
	"errors"
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

	// Metadata related to the segment files.
	segments diskQueueSegments

	// Metadata related to consumer acks / positions of the oldest remaining
	// frame.
	acks *diskQueueACKs

	// The queue's helper loops, each of which is run in its own goroutine.
	readerLoop  *readerLoop
	writerLoop  *writerLoop
	deleterLoop *deleterLoop

	// Wait group for shutdown of the goroutines associated with this queue:
	// reader loop, writer loop, deleter loop, and core loop (diskQueue.run()).
	waitGroup sync.WaitGroup

	// writing is true if the writer loop is processing a request, false
	// otherwise.
	writing bool

	// If writing is true, then writeRequestSize equals the number of bytes it
	// contained. Used to calculate how much free capacity the queue has left
	// after all scheduled writes have been completed (see canAcceptFrameOfSize).
	writeRequestSize uint64

	// reading is true if the reader loop is processing a request, false
	// otherwise.
	reading bool

	// deleting is true if the deleter loop is processing a request, false
	// otherwise.
	deleting bool

	// The API channel used by diskQueueProducer to write events.
	producerWriteRequestChan chan producerWriteRequest

	// pendingFrames is a list of all incoming data frames that have been
	// accepted by the queue and are waiting to be sent to the writer loop.
	// Segment ids in this list always appear in sorted order, even between
	// requests (that is, a frame added to this list always has segment id
	// at least as high as every previous frame that has ever been added).
	pendingFrames []segmentedFrame

	// blockedProducers is a list of all producer write requests that are
	// waiting for free space in the queue.
	blockedProducers []producerWriteRequest

	// The channel to signal our goroutines to shut down.
	done chan struct{}
}

func init() {
	queue.RegisterQueueType(
		"disk",
		queueFactory,
		feature.MakeDetails(
			"Disk queue",
			"Buffer events on disk before sending to the output.",
			feature.Stable))
}

// queueFactory matches the queue.Factory interface, and is used to add the
// disk queue to the registry.
func queueFactory(
	ackListener queue.ACKListener, logger *logp.Logger, cfg *common.Config, _ int, // input queue size param is unused.
) (queue.Queue, error) {
	settings, err := SettingsForUserConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("disk queue couldn't load user config: %w", err)
	}
	settings.WriteToDiskListener = ackListener
	return NewQueue(logger, settings)
}

// NewQueue returns a disk-based queue configured with the given logger
// and settings, creating it if it doesn't exist.
func NewQueue(logger *logp.Logger, settings Settings) (*diskQueue, error) {
	logger = logger.Named("diskqueue")
	logger.Debugf(
		"Initializing disk queue at path %v", settings.directoryPath())

	if settings.MaxBufferSize > 0 &&
		settings.MaxBufferSize < settings.MaxSegmentSize*2 {
		return nil, fmt.Errorf(
			"disk queue buffer size (%v) must be at least "+
				"twice the segment size (%v)",
			settings.MaxBufferSize, settings.MaxSegmentSize)
	}

	// Create the given directory path if it doesn't exist.
	err := os.MkdirAll(settings.directoryPath(), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("couldn't create disk queue directory: %w", err)
	}

	// Load the previous queue position, if any.
	nextReadPosition, err := queuePositionFromPath(settings.stateFilePath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// Errors reading / writing the position are non-fatal -- we just log a
		// warning and fall back on the oldest existing segment, if any.
		logger.Warnf("Couldn't load most recent queue position: %v", err)
	}
	if nextReadPosition.frameIndex == 0 {
		// If the previous state was written by an older version, it may lack
		// the frameIndex field. In this case we reset the read offset within
		// the segment, which may cause one-time retransmission of some events
		// from a previous version, but ensures that our metrics are consistent.
		// In the more common case that frameIndex is 0 because this segment
		// simply hasn't been read yet, setting byteIndex to 0 is a no-op.
		nextReadPosition.byteIndex = 0
	}
	positionFile, err := os.OpenFile(
		settings.stateFilePath(), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		// This is not the _worst_ error: we could try operating even without a
		// position file. But it indicates a problem with the queue permissions on
		// disk, which keeps us from tracking our position within the segment files
		// and could also prevent us from creating new ones, so we treat this as a
		// fatal error on startup rather than quietly providing degraded
		// performance.
		return nil, fmt.Errorf("couldn't write to state file: %w", err)
	}

	// Index any existing data segments to be placed in segments.reading.
	initialSegments, err :=
		scanExistingSegments(logger, settings.directoryPath())
	if err != nil {
		return nil, err
	}
	var nextSegmentID segmentID
	if len(initialSegments) > 0 {
		// Initialize nextSegmentID to the first ID after the existing segments.
		lastID := initialSegments[len(initialSegments)-1].id
		nextSegmentID = lastID + 1
	}

	// If any of the initial segments are older than the current queue
	// position, move them directly to the acked list where they can be
	// deleted.
	ackedSegments := []*queueSegment{}
	readSegmentID := nextReadPosition.segmentID
	for len(initialSegments) > 0 && initialSegments[0].id < readSegmentID {
		ackedSegments = append(ackedSegments, initialSegments[0])
		initialSegments = initialSegments[1:]
	}

	// If the queue position is older than all existing segments, advance
	// it to the beginning of the first one.
	if len(initialSegments) > 0 && readSegmentID < initialSegments[0].id {
		nextReadPosition = queuePosition{segmentID: initialSegments[0].id}
	}

	// We can compute the active frames right now but still need a way to report
	// them to the global beat metrics. For now, just log the total.
	// Note that for consistency with existing queue behavior, this excludes
	// events that are still present on disk but were already sent and
	// acknowledged on a previous run (we probably want to track these as well
	// in the future.)
	// TODO: pass in a context that queues can use to report these events. //nolint:godox //Ignore This
	activeFrameCount := 0
	for _, segment := range initialSegments {
		activeFrameCount += int(segment.frameCount)
	}
	activeFrameCount -= int(nextReadPosition.frameIndex)
	logger.Infof("Found %d existing events on queue start", activeFrameCount)

	queue := &diskQueue{
		logger:   logger,
		settings: settings,

		segments: diskQueueSegments{
			reading:          initialSegments,
			acked:            ackedSegments,
			nextID:           nextSegmentID,
			nextReadPosition: nextReadPosition.byteIndex,
		},

		acks: newDiskQueueACKs(logger, nextReadPosition, positionFile),

		readerLoop:  newReaderLoop(settings),
		writerLoop:  newWriterLoop(logger, settings),
		deleterLoop: newDeleterLoop(settings),

		producerWriteRequestChan: make(chan producerWriteRequest),

		done: make(chan struct{}),
	}

	// We wait for four goroutines on shutdown: core loop, reader loop,
	// writer loop, deleter loop.
	queue.waitGroup.Add(4)

	// Start the goroutines and return the queue!
	go func() {
		queue.readerLoop.run()
		queue.waitGroup.Done()
	}()
	go func() {
		queue.writerLoop.run()
		queue.waitGroup.Done()
	}()
	go func() {
		queue.deleterLoop.run()
		queue.waitGroup.Done()
	}()
	go func() {
		queue.run()
		queue.waitGroup.Done()
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
		done:    make(chan struct{}),
	}
}

func (dq *diskQueue) Consumer() queue.Consumer {
	return &diskQueueConsumer{queue: dq, done: make(chan struct{})}
}

func (dq *diskQueue) Metrics() (queue.Metrics, error) {
	return queue.Metrics{}, queue.ErrMetricsNotImplemented
}
