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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
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

	// A listener that receives ACKs when events are removed from the queue
	// and written to their output.
	// This can only be effective for events that are added to the queue
	// after it is opened (there is no way to acknowledge input from a
	// previous execution). It is ignored for events before that.
	//WriteToOutputACKListener queue.ACKListener
}

type segmentID uint64

type queuePosition struct {
	// The index of this position's segment within the queue.
	segment segmentID

	// The byte offset of this position within its segment.
	byteIndex segmentPos
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
	segments diskQueueSegments

	// The total bytes occupied by all segment files. This is the value
	// we check to see if there is enough space to add an incoming event
	// to the queue.
	bytesOnDisk uint64

	// The memory queue of data blobs waiting to be written to disk.
	// To add something to the queue internally, send it to this channel.
	inChan chan []byte

	outChan chan diskQueueOutput

	// The currently active segment reader, or nil if there is none.
	//reader *segmentReader

	// The currently active segment writer. When the corresponding segment
	// is full it is appended to segments.
	//writer *segmentWriter

	// The ol
	firstPosition bufferPosition

	// The position of the next event to read from the queue. If this equals
	// writePosition, then there are no events left to read.
	// This is initialized to state.firstPosition, but generally the two differ:
	// readPosition is advanced when an event is read, but firstPosition is
	// only advanced when the event has been read _and_ its consumer receives
	// an acknowledgement (meaning it has been transmitted and can be removed
	// from the queue).
	// This is part of diskQueue and not diskQueueState since it represents
	// in-memory state that should not persist through a restart.
	readPosition bufferPosition

	// A condition that is signalled when a segment file is deleted.
	// Used by writerLoop when the queue is full, to detect when to try again.
	// When the queue is closed, this condition will receive a broadcast after
	// diskQueue.closed is set to true.
	segmentDeletedCond sync.Cond

	// A condition that is signalled when a frame has been completely
	// written to disk.
	// Used by readerLoop when the queue is empty, to detect when to try again.
	// When the queue is closed, this condition will receive a broadcast after
	// diskQueue.closed is set to true.
	frameWrittenCond sync.Cond

	// The oldest frame id that is still stored on disk.
	// This will usually be less than ackedUpTo, since oldestFrame can't
	// advance until the entire segment file has been acknowledged and
	// deleted.
	oldestFrame frameID

	// This lock must be held to read and write acked and ackedUpTo.
	ackLock sync.Mutex

	// The lowest frame id that has not yet been acknowledged.
	ackedUpTo frameID

	// A map of all acked indices that are above ackedUpTo (and thus
	// can't yet be acknowledged as a continuous block).
	acked map[frameID]bool

	// Whether the queue has been closed. Code that can't check the done
	// channel (e.g. code that must wait on a condition variable) should
	// always check this value when waking up.
	closed atomic.Bool

	// The channel to signal our goroutines to shut down.
	done chan struct{}
}

// diskQueueSegments encapsulates segment-related queue metadata.
type diskQueueSegments struct {
	// The lock should be held to read or write any of the fields below.
	sync.Mutex

	// The segment that is currently being written.
	writing *queueSegment

	writer *segmentWriter
	reader *segmentReader

	// A list of the segments that have been completely written but have
	// not yet been processed by the reader loop, sorted by increasing
	// segment ID. Segments are always read in order. When a segment has
	// been read completely, it is removed from the front of this list and
	// appended to completedSegments.
	reading []*queueSegment

	// A list of the segments that have been read but have not yet been
	// completely acknowledged, sorted by increasing segment ID. When the
	// first entry of this list is completely acknowledged, it is removed
	// from this list and the underlying file is deleted.
	completed []*queueSegment

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
	// Create the given directory path if it doesn't exist.
	err := os.MkdirAll(settings.Path, os.ModePerm)
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

	segments, err := queueSegmentsForPath(
		settings.directoryPath(), settings.Logger)
	if err != nil {
		return nil, err
	}

	return &diskQueue{
		settings: settings,
		segments: diskQueueSegments{
			reading: segments,
		},
		closed: atomic.MakeBool(false),
		done:   make(chan struct{}),
	}, nil
}

// This is only called by readerLoop.
func (dq *diskQueue) nextSegmentReader() (*segmentReader, []error) {
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
		dq.segments.completed = append(dq.segments.completed, segment)
		return reader, errors
	}
	// TODO: if segments.reading is empty we may still be able to
	// read partial data from segments.writing which is still being
	// written.
	return nil, errors
}

func tryLoad(segment *queueSegment, path string) (*segmentReader, error) {
	// this is a strangely fine-grained lock maybe?
	segment.Lock()
	defer segment.Unlock()

	// dataSize is guaranteed to be positive because we don't add
	// anything to the segments list unless it is.
	dataSize := segment.size - segmentHeaderSize
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't open segment %d: %w", segment.id, err)
	}
	reader := bufio.NewReader(file)
	header, err := readSegmentHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read segment header: %w", err)
	}
	return &segmentReader{
		raw:          reader,
		curPosition:  0,
		endPosition:  segmentPos(dataSize),
		checksumType: header.checksumType,
	}, nil
}

type segmentHeader struct {
	version      uint32
	checksumType checksumType
}

func readSegmentHeader(in io.Reader) (*segmentHeader, error) {
	header := segmentHeader{}
	if header.version != 0 {
		return nil, fmt.Errorf("Unrecognized schema version %d", header.version)
	}
	panic("TODO: not implemented")
	//return nil, nil
}

// readNextFrame reads the next pending data frame in the queue
// and returns its contents.
/*func (dq *diskQueue) readNextFrame() ([]byte, error) {
	// READER LOCK --->
	if dq.reader != nil {
		frameData, err := dq.reader.nextDataFrame()
		if err != nil {
			return nil, err
		}
		if frameData != nil {
			return frameData, nil
		}
		// If we made it here then the active reader was empty and
		// we need to fetch a new one.
	}
	reader, _ := dq.nextSegmentReader()
	dq.reader = reader
	return reader.nextDataFrame()
	// <--- READER LOCK
}*/

//
// bookkeeping helpers to locate queue data on disk
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

//
// diskQueue implementation of the queue.Queue interface
//

func (dq *diskQueue) Close() error {
	if dq.closed.Swap(true) {
		return fmt.Errorf("Can't close disk queue: queue already closed")
	}
	// TODO: wait for worker threads?
	close(dq.done)
	return nil
}

func (dq *diskQueue) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{MaxEvents: 0}
}

func (dq *diskQueue) Producer(cfg queue.ProducerConfig) queue.Producer {
	panic("TODO: not implemented")
}

func (dq *diskQueue) Consumer() queue.Consumer {
	panic("TODO: not implemented")
}
