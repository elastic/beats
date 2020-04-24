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
	// is stored in queue.dat and each segment's data is stored in
	// segment.{segmentIndex}
	// If blank, the default directory is "diskqueue" within the beat's data
	// directory.
	Path string

	// The size in bytes of one data page in the on-disk buffer. To minimize
	// data loss if there is an error, this should match the page size of the
	// target filesystem.
	PageSize uint32

	// MaxBufferSize is the maximum number of bytes that the queue should
	// ever occupy on disk. A value of 0 means the queue can grow until the
	// disk is full.
	MaxBufferSize uint64

	// A listener that receives ACKs when events are written to the queue's
	// disk buffer.
	WriteToDiskACKListener queue.ACKListener

	// A listener that receives ACKs when events are removed from the queue
	// and written to their output.
	WriteToOutputACKListener queue.ACKListener
}

type bufferPosition struct {
	// The segment index of this position within the overall buffer.
	segmentIndex uint64

	// The page index of this position within its segment.
	pageIndex uint64

	// The byte index of this position within its page's data region.
	byteIndex uint32
}

type diskQueueState struct {
	// The page size of the queue. This is originally derived from
	// Settings.PageSize, and the two must match during normal queue operation.
	// They can only differ during data recovery / page size migration.
	pageSize uint32

	// The oldest position in the queue. This is advanced as we receive ACKs from
	// downstream consumers indicating it is safe to remove old events.
	firstPosition bufferPosition

	// The position of the next (unwritten) byte in the queue buffer. When an
	// event is added to the queue, this position is advanced to point to the
	// first byte after its end.
	lastPosition bufferPosition

	// The maximum number of pages that can be used for the queue buffer.
	// This is derived by dividing Settings.MaxBufferSize by pageSize and
	// rounding down.
	maxPageCount uint64

	// The number of pages currently occupied by the queue buffer. This can't
	// be derived from firstPosition and lastPosition because segment length
	// varies with the size of their last event.
	// This can be greater than maxPageCount if the maximum buffer size is
	// reduced on an already-full queue.
	allocatedPageCount uint64
}

// diskQueue is the internal type representing a disk-based implementation
// of queue.Queue.
type diskQueue struct {
	settings Settings

	// The persistent queue state. After a filesystem sync this should be
	// identical to the queue's metadata file.
	state diskQueueState

	// The position of the next event to read from the queue. If this equals
	// state.lastPosition, then there are no events left to read.
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
// and settings.
func NewQueue(settings Settings) (queue.Queue, error) {
	if settings.Path == "" {
		settings.Path = paths.Resolve(paths.Data, "queue.dat")
	}
	return &diskQueue{settings: settings}, nil
}

//
// diskQueue mplementation of the queue.Queue interface
//

func (dq *diskQueue) Close() error {
	panic("TODO: not implemented")
}

func (dq *diskQueue) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{MaxEvents: 0}
}

func (dq *diskQueue) Producer(cfg queue.ProducerConfig) queue.Producer {
	/*return &diskQueueProducer{
		queue:  dq,
		config: cfg,
	}*/
	panic("TODO: not implemented")
}

func (dq *diskQueue) Consumer() queue.Consumer {
	panic("TODO: not implemented")
}
