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

	"github.com/elastic/beats/v7/libbeat/common"
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

type bufferPosition struct {
	// The index of this position's segment within the overall buffer.
	segment segmentID

	// The byte offset of this position within its segment.
	byteIndex uint64
}

type diskQueueOutput struct {
	data []byte

	// The segment file this data was read from.
	segment *segmentFile

	// The index of this data's frame within its segment.
	frameIndex int
}

// diskQueue is the internal type representing a disk-based implementation
// of queue.Queue.
type diskQueue struct {
	settings Settings

	// The persistent queue state (wraps diskQueuePersistentState on disk).
	stateFile *stateFile

	// A list of all segments that have been completely written but have
	// not yet been handed off to a segmentReader.
	// Sorted by increasing segment ID.
	segments []segmentFile

	// The total bytes occupied by all segment files. This is the value
	// we check to see if there is enough space to add an incoming event
	// to the queue.
	bytesOnDisk uint64

	// The memory queue of data blobs waiting to be written to disk.
	// To add something to the queue internally, send it to this channel.
	inChan chan byte[]

	outChan chan diskQueueOutput

	// The currently active segment reader, or nil if there is none.
	reader *segmentReader

	// The currently active segment writer. When the corresponding segment
	// is full it is appended to segments.
	writer *segmentWriter

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

	segments, err := segmentFilesForPath(settings.directoryPath())
	if err != nil {
		return nil, err
	}

	return &diskQueue{
		settings: settings,
		segments: segments,
	}, nil
}

func (dq *diskQueue) nextSegmentReader() (*segmentReader, error) {
	if len(dq.segments) > 0 {
		return nil, nil
	}
	nextSegment := dq.segments[0]

}

// readNextFrame reads the next pending data frame in the queue
// and returns its contents.
func (dq *diskQueue) readNextFrame() ([]byte, error) {
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
	reader, err := dq.nextSegmentReader()
	dq.reader = reader
	return reader.nextDataFrame()
	// <--- READER LOCK
}

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

func (settings Settings) segmentFilePath(segmentID uint64) string {
	return filepath.Join(
		settings.directoryPath(),
		fmt.Sprintf("%v.seg", segmentID))
}

//
// diskQueue implementation of the queue.Queue interface
//

func (dq *diskQueue) Close() error {
	panic("TODO: not implemented")
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
