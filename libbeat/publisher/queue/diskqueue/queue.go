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
	"path/filepath"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// Settings contains the configuration fields to create a new disk queue
// or open an existing one.
type Settings struct {
	// The destination for log messages related to the disk queue.
	Logger *logp.Logger

	// The path on disk of the queue's containing directory, which will be
	// created if it doesn't exist. Within the directory, the queue's state
	// is stored in state.dat and each segment's data is stored in
	// {segmentIndex}.seg
	// If blank, the default directory is "diskqueue" within the beat's data
	// directory.
	Path string

	// MaxBufferSize is the maximum number of bytes that the queue should
	// ever occupy on disk. A value of 0 means the queue can grow until the
	// disk is full (this is not recommended on a primary system disk).
	MaxBufferSize uint64

	// MaxSegmentSize is the maximum number of bytes that should be written
	// to a single segment file before creating a new one.
	MaxSegmentSize uint64

	// A listener that receives ACKs when events are written to the queue's
	// disk buffer.
	WriteToDiskACKListener queue.ACKListener

	ChecksumType ChecksumType
}

type segmentID uint64

type queuePosition struct {
	// The index of this position's segment within the queue.
	segment segmentID

	// The byte offset of this position within its segment.
	// This is specified relative to the start of the segment's data region, i.e.
	// an offset of 0 means the first byte after the end of the segment header.
	offset segmentOffset
}

type diskQueueOutput struct {
	data []byte

	// The segment file this data was read from.
	segment *queueSegment

	// The index of this data's frame (the sequential read order
	// of all frames during this execution).
	frame frameID
}

// diskQueue is the internal type representing a disk-based implementation
// of queue.Queue.
type diskQueue struct {
	settings Settings

	// The persistent queue state (wraps diskQueuePersistentState on disk).
	stateFile *stateFile

	// Metadata related to the segment files.
	segments *diskQueueSegments

	coreLoop    *coreLoop
	readerLoop  *readerLoop
	writerLoop  *writerLoop
	deleterLoop *deleterLoop

	// The API channels used by diskQueueProducer to send write / cancel calls.
	producerWriteRequestChan  chan *producerWriteRequest
	producerCancelRequestChan chan *producerCancelRequest

	// When a consumer ack increments ackedUpTo, the consumer sends
	// its new value to this channel. The core loop then decides whether to
	// delete the containing segments.
	// The value sent on the channel is redundant with the value of ackedUpTo,
	// but we send it anyway so we don't have to worry about the core loop
	// waiting on ackLock.
	consumerAckChan chan frameID

	// This lock must be held to read and write acked and ackedUpTo.
	ackLock sync.Mutex

	// The lowest frame id that has not yet been acknowledged.
	ackedUpTo frameID

	// A map of all acked indices that are above ackedUpTo (and thus
	// can't yet be acknowledged as a continuous block).
	// TODO: do this better.
	acked map[frameID]bool

	// Wait group for shutdown of the goroutines associated with this queue:
	// core loop, reader loop, writer loop, deleter loop.
	waitGroup *sync.WaitGroup

	// The channel to signal our goroutines to shut down.
	done chan struct{}
}

// pendingFrame stores a single incoming event waiting to be written to disk,
// along with its serialization and metadata needed to notify its originating
// producer of ack / cancel state.
type pendingFrame struct {
	event    publisher.Event
	producer *diskQueueProducer
}

// pendingFrameData stores data frames waiting to be written to disk, with
// metadata to handle acks / cancellation if needed.
type pendingFrameData struct {
	sync.Mutex

	frames []pendingFrame
}

// diskQueueSegments encapsulates segment-related queue metadata.
type diskQueueSegments struct {
	// The segments that are currently being written. The writer loop
	// writes these segments in order. When a segment has been
	// completely written, the writer loop notifies the core loop
	// in a writeResponse, and it is moved to the reading list.
	// If the reading list is empty, the reader loop may read from
	// a segment that is still being written, but it will always
	// be writing[0], since later entries have generally not been
	// created yet.
	writing []*queueSegment

	// A list of the segments that have been completely written but have
	// not yet been completely processed by the reader loop, sorted by increasing
	// segment ID. Segments are always read in order. When a segment has
	// been read completely, it is removed from the front of this list and
	// appended to read.
	reading []*queueSegment

	// A list of the segments that have been read but have not yet been
	// completely acknowledged, sorted by increasing segment ID. When the
	// first entry of this list is completely acknowledged, it is removed
	// from this list and added to acked.
	acking []*queueSegment

	// A list of the segments that have been completely processed and are
	// ready to be deleted. The writer loop always tries to delete segments
	// in this list before writing new data. When a segment is successfully
	// deleted, it is removed from this list and the queue's
	// segmentDeletedCond is signalled.
	acked []*queueSegment

	// The next sequential unused segment ID. This is what will be assigned
	// to the next queueSegment we create.
	nextID segmentID
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

// queueFactory matches the queue.Factory type, and is used to add the disk
// queue to the registry.
func queueFactory(
	ackListener queue.ACKListener, logger *logp.Logger, cfg *common.Config,
) (queue.Queue, error) {
	settings, err := SettingsForUserConfig(cfg)
	if err != nil {
		return nil, err
	}
	settings.Logger = logger
	// For now, incoming messages are acked when they are written to disk
	// (rather than transmitted to the output, as with the memory queue). This
	// can produce unexpected behavior in some contexts and we might want to
	// make it configurable later.
	settings.WriteToDiskACKListener = ackListener
	return NewQueue(settings)
}

// NewQueue returns a disk-based queue configured with the given logger
// and settings, creating it if it doesn't exist.
func NewQueue(settings Settings) (queue.Queue, error) {
	settings.Logger.Debugf(
		"Initializing disk queue at path %v", settings.directoryPath())

	// Create the given directory path if it doesn't exist.
	err := os.MkdirAll(settings.directoryPath(), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("Couldn't create disk queue directory: %w", err)
	}

	// Load the file handle for the queue state.
	stateFile, err := stateFileForPath(settings.stateFilePath())
	if err != nil {
		return nil, fmt.Errorf("Couldn't open disk queue metadata file: %w", err)
	}
	defer func() {
		if err != nil {
			// If the function is returning because of an error, close the
			// file handle.
			stateFile.Close()
		}
	}()

	initialSegments, err := scanExistingSegments(settings.directoryPath())
	if err != nil {
		return nil, err
	}

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
		requestChan:  make(chan readRequest, 1),
		responseChan: make(chan readResponse),
		output:       make(chan *readFrame, 20), // TODO: customize this buffer size
	}
	go func() {
		readerLoop.run()
		waitGroup.Done()
	}()

	writerLoop := &writerLoop{
		logger:       settings.Logger,
		requestChan:  make(chan writeRequest, 1),
		responseChan: make(chan writeResponse),
	}
	go func() {
		writerLoop.run()
		waitGroup.Done()
	}()

	deleterLoop := &deleterLoop{
		queueSettings: &settings,
		input:         make(chan *deleteRequest),
		response:      make(chan *deleteResponse),
	}
	go func() {
		deleterLoop.run()
		waitGroup.Done()
	}()

	queue := &diskQueue{
		settings:  settings,
		stateFile: stateFile,
		segments: &diskQueueSegments{
			reading: initialSegments,
		},
		readerLoop:  readerLoop,
		writerLoop:  writerLoop,
		deleterLoop: deleterLoop,
		waitGroup:   &waitGroup,
		done:        make(chan struct{}),
	}

	// The core loop is created last because it's the only one that needs
	// to refer back to the queue. (TODO: just merge the core loop fields
	// and logic into the queue itself.)
	queue.coreLoop = &coreLoop{
		queue:          queue,
		nextReadOffset: 0, // TODO: initialize this if we're opening an existing queue
	}
	go func() {
		queue.coreLoop.run()
		waitGroup.Done()
	}()

	return queue, nil
}

//
// bookkeeping helpers
//

func (settings Settings) directoryPath() string {
	if settings.Path == "" {
		return paths.Resolve(paths.Data, "diskqueue")
	}

	return settings.Path
}

func (settings Settings) stateFilePath() string {
	return filepath.Join(settings.directoryPath(), "state.dat")
}

func (settings Settings) segmentPath(segmentID segmentID) string {
	return filepath.Join(
		settings.directoryPath(),
		fmt.Sprintf("%v.seg", segmentID))
}

func (settings Settings) maxSegmentOffset() segmentOffset {
	return segmentOffset(settings.MaxSegmentSize - segmentHeaderSize)
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
		encoder: newFrameEncoder(dq.settings.ChecksumType),
	}
}

func (dq *diskQueue) Consumer() queue.Consumer {
	panic("TODO: not implemented")
}

// This is only called by readerLoop.
/*func (dq *diskQueue) nextSegmentReader() (*segmentReader, []error) {
	dq.segments.Lock()
	defer dq.segments.Unlock()

	errors := []error{}
	for len(dq.segments.reading) > 0 {
		segment := dq.segments.reading[0]
		segmentPath := dq.settings.segmentPath(segment.id)
		reader, err := tryLoad(segment, segmentPath)
		if err != nil {
			// TODO: Handle this: depending on the type of error, either delete
			// the segment or log an error and leave it alone, then skip to the
			// next one.
			errors = append(errors, err)
			dq.segments.reading = dq.segments.reading[1:]
			continue
		}
		// Remove the segment from the active list and move it to
		// completedSegments until all its data has been acknowledged.
		dq.segments.reading = dq.segments.reading[1:]
		dq.segments.acking = append(dq.segments.acking, segment)
		return reader, errors
	}
	// TODO: if segments.reading is empty we may still be able to
	// read partial data from segments.writing which is still being
	// written.
	return nil, errors
}*/
