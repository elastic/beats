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

import "github.com/elastic/beats/v7/libbeat/publisher/queue"

// Every data frame read from the queue is assigned a unique sequential
// integer, which is used to keep track of which frames have been
// acknowledged.
// This id is not stable between restarts; the value 0 is always assigned
// to the oldest remaining frame on startup.
type frameID uint64

// A data frame created through the producer API and waiting to be
// written to disk.
type writeFrame struct {
	// The event, serialized for writing to disk and wrapped in a frame
	// header / footer.
	serialized []byte

	// The producer that created this frame. This is included in the
	// frame structure itself because we may need the producer and / or
	// its config at any time up until it has been completely written:
	// - While the core loop is tracking frames to send to the writer,
	//   it may receive a Cancel request, which requires us to know
	//   the producer / config each frame came from.
	// - After the writer loop has finished writing the frame to disk,
	//   it needs to call the ACK function specified in ProducerConfig.
	producer *diskQueueProducer
}

// A frame that has been read from disk and is waiting to be read /
// acknowledged through the consumer API.
type readFrame struct {
	// The segment containing this frame.
	segment *queueSegment

	// The id of this frame.
	id frameID

	// The event decoded from the data frame.
	event queue.Event

	// How much space this frame occupied on disk (before deserialization),
	// including the frame header / footer.
	bytesOnDisk uint64
}

// Each data frame has a 32-bit length in the header, and a 32-bit checksum
// and a duplicate 32-bit length in the footer.
const frameHeaderSize = 4
const frameFooterSize = 8
const frameMetadataSize = frameHeaderSize + frameFooterSize

func (frame writeFrame) sizeOnDisk() uint64 {
	return uint64(len(frame.serialized) + frameMetadataSize)
}
