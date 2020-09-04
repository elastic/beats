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
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Every data frame read from the queue is assigned a unique sequential
// integer, which is used to keep track of which frames have been
// acknowledged.
// This id is not stable between restarts; the value 0 is always assigned
// to the oldest remaining frame on startup.
type frameID uint64

// segmentOffset is a byte index into the segment's data region.
// An offset of 0 means the first byte after the segment file header.
type segmentOffset uint64

// The metadata for a single segment file.
type queueSegment struct {
	// A segment id is globally unique within its originating queue.
	id segmentID

	// The settings for the queue that created this segment. Used for locating
	// the queue file on disk and determining its checksum behavior.
	queueSettings *Settings

	// Whether the file for this segment exists on disk yet. If it does
	// not, then calling getWriter() will create it and return a writer
	// positioned at the start of the data region.
	created bool

	// The byte offset of the end of the segment's data region. This is
	// updated when the segment is written to, and should always correspond
	// to the end of a complete data frame. The total size of a segment file
	// on disk is segmentHeaderSize + segment.endOffset.
	endOffset segmentOffset

	// The header metadata for this segment file. This field is nil if the
	// segment has not yet been opened for reading. It should only be
	// accessed by the reader loop.
	header *segmentHeader

	// The number of frames read from this segment during this session. This
	// does not necessarily equal the number of frames in the segment, even
	// after reading is complete, since the segment may have been partially
	// read during a previous session.
	//
	// Used to count how many frames still need to be acknowledged by consumers.
	framesRead int64
}

type segmentHeader struct {
	version      uint32
	checksumType ChecksumType
}

// Each data frame has a 32-bit length and a 32-bit checksum
// in the header, and a duplicate 32-bit length in the footer.
const frameHeaderSize = 8
const frameFooterSize = 4
const frameMetadataSize = frameHeaderSize + frameFooterSize

// Each segment header has a 32-bit version and a 32-bit checksum type.
const segmentHeaderSize = 8

// Sort order: we store loaded segments in ascending order by their id.
type bySegmentID []*queueSegment

func (s bySegmentID) Len() int           { return len(s) }
func (s bySegmentID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s bySegmentID) Less(i, j int) bool { return s[i].endOffset < s[j].endOffset }

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
						created:   true,
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
func (segment *queueSegment) getReader() (*os.File, error) {
	path := segment.queueSettings.segmentPath(segment.id)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't open segment %d: %w", segment.id, err)
	}
	header, err := readSegmentHeader(file)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read segment header: %w", err)
	}
	segment.header = header

	return file, nil
}

// Should only be called from the writer loop.
func (segment *queueSegment) getWriter() (io.WriteCloser, error) {
	path := segment.queueSettings.segmentPath(segment.id)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't open segment %d: %w", segment.id, err)
	}
	header := &segmentHeader{
		version:      0,
		checksumType: segment.queueSettings.ChecksumType,
	}
	err = writeSegmentHeader(file, header)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't write to new segment %d: %w", segment.id, err)
	}
	return file, nil
}

func readSegmentHeader(in io.Reader) (*segmentHeader, error) {
	header := segmentHeader{}
	if header.version != 0 {
		return nil, fmt.Errorf("Unrecognized schema version %d", header.version)
	}
	panic("TODO: not implemented")
	//return nil, nil
}

func writeSegmentHeader(out io.Writer, header *segmentHeader) error {
	panic("TODO: not implemented")
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

// nextDataFrame returns the bytes of the next data frame, or an error if the
// frame couldn't be read. If an error is returned, the caller should log it
// and drop the containing segment. A nil return value with no error means
// there are no frames to read.
/*func (reader *segmentReader) nextDataFrame() ([]byte, error) {
	if reader.curPosition >= reader.endPosition {
		return nil, nil
	}
	var frameLength uint32
	err := binary.Read(reader.raw, binary.LittleEndian, &frameLength)
	if err != nil {
		return nil, fmt.Errorf(
			"Disk queue couldn't read next frame length: %w", err)
	}

	// Bounds checking to make sure we can read this frame.
	if reader.curPosition+segmentOffset(frameLength) > reader.endPosition {
		// This frame extends past the end of our data region, which
		// should never happen unless there is data corruption.
		return nil, fmt.Errorf(
			"Data frame length (%d) exceeds remaining data (%d)",
			frameLength, reader.endPosition-reader.curPosition)
	}
	if frameLength <= frameMetadataSize {
		// Actual enqueued data must have positive length
		return nil, fmt.Errorf(
			"Data frame with no data (length %d)", frameLength)
	}

	// Read the actual frame data
	dataLength := frameLength - frameMetadataSize
	data := make([]byte, dataLength)
	_, err = io.ReadFull(reader.raw, data)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't read data frame from disk: %w", err)
	}

	// Read the footer (length + checksum)
	var duplicateLength uint32
	err = binary.Read(reader.raw, binary.LittleEndian, &duplicateLength)
	if err != nil {
		return nil, fmt.Errorf(
			"Disk queue couldn't read trailing frame length: %w", err)
	}
	if duplicateLength != frameLength {
		return nil, fmt.Errorf(
			"Disk queue: inconsistent frame length (%d vs %d)",
			frameLength, duplicateLength)
	}

	// Validate the checksum
	var checksum uint32
	err = binary.Read(reader.raw, binary.LittleEndian, &checksum)
	if err != nil {
		return nil, fmt.Errorf(
			"Disk queue couldn't read data frame's checksum: %w", err)
	}
	if computeChecksum(data, reader.checksumType) != checksum {
		return nil, fmt.Errorf("Disk queue: bad data frame checksum")
	}

	reader.curPosition += segmentOffset(frameLength)
	return data, nil
}*/
