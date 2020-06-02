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
	WriteToOutputACKListener queue.ACKListener
}

type segmentID uint64

type bufferPosition struct {
	// The index of this position's segment within the overall buffer.
	segment segmentID

	// The byte offset of this position within its segment.
	byteIndex uint64
}

// diskQueue is the internal type representing a disk-based implementation
// of queue.Queue.
type diskQueue struct {
	settings Settings

	// The persistent queue state (wraps diskQueuePersistentState on disk).
	stateFile *stateFile

	segments *segmentManager

	//
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

	return &diskQueue{settings: settings}, nil
}

func (dq *diskQueue) getSegment(id segmentID) (*segmentFile, error) {
	panic("TODO: not implemented")
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
