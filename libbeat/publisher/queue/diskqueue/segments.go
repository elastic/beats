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
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

// diskQueueSegments encapsulates segment-related queue metadata.
type diskQueueSegments struct {

	// A list of the segments that have not yet been completely written, sorted
	// by increasing segment ID. When the first entry has been completely
	// written, it is removed from this list and appended to reading.
	//
	// If the reading list is empty, the queue may read from a segment that is
	// still being written, but it will always be writing[0], since later
	// entries do not yet exist on disk.
	writing []*queueSegment

	// A list of the segments that have been completely written but have
	// not yet been completely read, sorted by increasing segment ID. When the
	// first entry has been completely read, it is removed from this list and
	// appended to acking.
	reading []*queueSegment

	// A list of the segments that have been completely read but have not yet
	// been completely acknowledged, sorted by increasing segment ID. When the
	// first entry has been completely acknowledged, it is removed from this
	// list and appended to acked.
	acking []*queueSegment

	// A list of the segments that have been completely read and acknowledged
	// and are ready to be deleted. When a segment is successfully deleted, it
	// is removed from this list and discarded.
	acked []*queueSegment

	// The next sequential unused segment ID. This is what will be assigned
	// to the next queueSegment we create.
	nextID segmentID

	// nextWriteOffset is the segment offset at which the next new frame
	// should be written. This offset always applies to the last entry of
	// writing[]. This is distinct from the endOffset field within a segment:
	// endOffset tracks how much data _has_ been written to a segment, while
	// nextWriteOffset also includes everything that is _scheduled_ to be
	// written.
	nextWriteOffset segmentOffset

	// nextReadFrameID is the first frame ID in the current or pending
	// read request.
	nextReadFrameID frameID

	// nextReadOffset is the segment offset corresponding to the frame
	// nextReadFrameID. This offset always applies to the first reading
	// segment: either reading[0], or writing[0] if reading is empty.
	nextReadOffset segmentOffset
}

// segmentID is a unique persistent integer id assigned to each created
// segment in ascending order.
type segmentID uint64

// segmentOffset is a byte index into the segment's data region.
// An offset of 0 means the first byte after the segment file header.
type segmentOffset uint64

// The metadata for a single segment file.
type queueSegment struct {
	// A segment id is globally unique within its originating queue.
	id segmentID

	// The byte offset of the end of the segment's data region. This is
	// updated when the segment is written to, and should always correspond
	// to the end of a complete data frame. The total size of a segment file
	// on disk is segmentHeaderSize + segment.endOffset.
	endOffset segmentOffset

	// The ID of the first frame that was / will be read from this segment.
	// This field is only valid after a read request has been sent for
	// this segment. (Currently it is only used to handle consumer ACKs,
	// which can only happen after reading has begun on the segment.)
	firstFrameID frameID

	// The number of frames read from this segment during this session. This
	// does not necessarily equal the number of frames in the segment, even
	// after reading is complete, since the segment may have been partially
	// read during a previous session.
	//
	// Used to count how many frames still need to be acknowledged by consumers.
	framesRead uint64
}

type segmentHeader struct {
	version uint32
}

// Segment headers are currently just a 32-bit version.
const segmentHeaderSize = 4

// Sort order: we store loaded segments in ascending order by their id.
type bySegmentID []*queueSegment

func (s bySegmentID) Len() int           { return len(s) }
func (s bySegmentID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s bySegmentID) Less(i, j int) bool { return s[i].id < s[j].id }

// Scan the given path for segment files, and return them in a list
// ordered by segment id.
func scanExistingSegments(path string) ([]*queueSegment, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read queue directory '%s': %w", path, err)
	}

	segments := []*queueSegment{}
	for _, file := range files {
		if file.Size() <= segmentHeaderSize {
			// Ignore segments that don't have at least some data beyond the
			// header (this will always be true of segments we write unless there
			// is an error).
			continue
		}
		components := strings.Split(file.Name(), ".")
		if len(components) == 2 && strings.ToLower(components[1]) == "seg" {
			// Parse the id as base-10 64-bit unsigned int. We ignore file names that
			// don't match the "[uint64].seg" pattern.
			if id, err := strconv.ParseUint(components[0], 10, 64); err == nil {
				segments = append(segments,
					&queueSegment{
						id:        segmentID(id),
						endOffset: segmentOffset(file.Size() - segmentHeaderSize),
					})
			}
		}
	}
	sort.Sort(bySegmentID(segments))
	return segments, nil
}

func (segment *queueSegment) sizeOnDisk() uint64 {
	return uint64(segment.endOffset) + segmentHeaderSize
}

// Should only be called from the reader loop.
func (segment *queueSegment) getReader(
	queueSettings Settings,
) (*os.File, error) {
	path := queueSettings.segmentPath(segment.id)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't open segment %d: %w", segment.id, err)
	}
	// Right now there is only one valid header (indicating schema version
	// zero) so we don't need the value itself.
	_, err = readSegmentHeader(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("Couldn't read segment header: %w", err)
	}

	return file, nil
}

// Should only be called from the writer loop.
func (segment *queueSegment) getWriter(
	queueSettings Settings,
) (*os.File, error) {
	path := queueSettings.segmentPath(segment.id)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	header := &segmentHeader{version: 0}
	err = writeSegmentHeader(file, header)
	if err != nil {
		return nil, fmt.Errorf("Couldn't write segment header: %w", err)
	}

	return file, nil
}

// getWriterWithRetry tries to create a file handle for writing via
// queueSegment.getWriter. On error, it retries as long as the given
// retry callback returns true. This is used for timed retries when
// creating a queue segment from the writer loop.
func (segment *queueSegment) getWriterWithRetry(
	queueSettings Settings, retry func(err error, firstTime bool) bool,
) (*os.File, error) {
	firstTime := true
	file, err := segment.getWriter(queueSettings)
	for err != nil && retry(err, firstTime) {
		// Set firstTime to false so the retry callback can perform backoff
		// etc if needed.
		firstTime = false

		// Try again
		file, err = segment.getWriter(queueSettings)
	}
	return file, err
}

func readSegmentHeader(in *os.File) (*segmentHeader, error) {
	header := &segmentHeader{}
	err := binary.Read(in, binary.LittleEndian, &header.version)
	if err != nil {
		return nil, err
	}
	if header.version != 0 {
		return nil, fmt.Errorf("Unrecognized schema version %d", header.version)
	}
	return header, nil
}

func writeSegmentHeader(out *os.File, header *segmentHeader) error {
	err := binary.Write(out, binary.LittleEndian, header.version)
	return err
}

// The number of bytes occupied by all the queue's segment files. This
// should only be called from the core loop.
func (segments *diskQueueSegments) sizeOnDisk() uint64 {
	total := uint64(0)
	for _, segment := range segments.writing {
		total += segment.sizeOnDisk()
	}
	for _, segment := range segments.reading {
		total += segment.sizeOnDisk()
	}
	for _, segment := range segments.acking {
		total += segment.sizeOnDisk()
	}
	for _, segment := range segments.acked {
		total += segment.sizeOnDisk()
	}
	return total
}
