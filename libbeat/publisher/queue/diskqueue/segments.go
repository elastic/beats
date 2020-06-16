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
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Every data frame read from the queue is assigned a unique sequential
// integer, which is used to keep track of which frames have been
// acknowledged.
type frameID uint64

// The metadata for a single segment file.
type queueSegment struct {
	sync.Mutex

	id segmentID

	// The length in bytes of the segment file on disk. This is updated when
	// the segment is written to, and should always correspond to the end of
	// a complete data frame.
	size uint64

	// The number of frames read from this segment, or zero if it has not
	// yet been completely read.
	// It is safe to delete a segment when framesRead > 0 and all those
	// frames have been acknowledged.
	framesRead int64
}

// segmentReader is a wrapper around io.Reader that provides helpers and
// metadata for decoding segment files.
type segmentReader struct {
	// The segment this reader was generated from.
	segment *queueSegment

	// The underlying data reader.
	raw io.Reader

	// The current byte offset of the reader within the file.
	curPosition int64

	// The position at which this reader should stop reading. This is often
	// the end of the file, but it may be earlier when the queue is reading
	// and writing to the same segment.
	endPosition int64

	// The checksumType field from this segment file's header.
	checksumType checksumType
}

type segmentWriter struct {
	segment  *queueSegment
	file     *os.File
	position int64
}

type checksumType int

const (
	checksumTypeNone = iota
	checksumTypeCRC32
)

// Each data frame has 2 32-bit lengths and 1 32-bit checksum.
const frameMetadataSize = 12

// Each segment header has a 32-bit version and a 32-bit checksum type.
const segmentHeaderSize = 8

// Sort order: we store loaded segments in ascending order by their id.
type bySegmentID []*queueSegment

func (s bySegmentID) Len() int           { return len(s) }
func (s bySegmentID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s bySegmentID) Less(i, j int) bool { return s[i].size < s[j].size }

func queueSegmentsForPath(
	path string, logger *logp.Logger,
) ([]*queueSegment, error) {
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
						id:   segmentID(id),
						size: uint64(file.Size()),
					})
			}
		}
	}
	sort.Sort(bySegmentID(segments))
	return segments, nil
}

// A nil data frame with no error means this reader has no more frames.
// If nextDataFrame returns an error, it should be logged and the
// corresponding segment should be dropped.
func (reader *segmentReader) nextDataFrame() ([]byte, error) {
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
	if reader.curPosition+int64(frameLength) > reader.endPosition {
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

	reader.curPosition += int64(frameLength)
	return data, nil
}

// returns the number of indices by which ackedUpTo was advanced.
func (dq *diskQueue) ack(frame frameID) int {
	dq.ackLock.Lock()
	defer dq.ackLock.Unlock()
	dq.acked[frame] = true
	ackedCount := 0
	for ; dq.acked[dq.ackedUpTo]; dq.ackedUpTo++ {
		delete(dq.acked, dq.ackedUpTo)
		ackedCount++
	}
	return ackedCount
}

func computeChecksum(data []byte, checksumType checksumType) uint32 {
	switch checksumType {
	case checksumTypeNone:
		return 0
	case checksumTypeCRC32:
		hash := crc32.NewIEEE()
		frameLength := uint32(len(data) + frameMetadataSize)
		binary.Write(hash, binary.LittleEndian, &frameLength)
		hash.Write(data)
		return hash.Sum32()
	default:
		panic("segmentReader: invalid checksum type")
	}
}

func (dq *diskQueue) segmentReaderForPosition(
	pos bufferPosition,
) (*segmentReader, error) {
	panic("TODO: not implemented")
}

/*
func (sm *segmentManager) segmentReaderForPosition(pos bufferPosition) (*segmentReader, error) {
	segment = getSegment(pos.segment)

	dataSize := segment.size - segmentHeaderSize
	file, err := os.Open(pathForSegmentId(pos.segment))
	// ...read segment header...
	checksumType := checksumTypeNone
	file.Seek(segmentHeaderSize+pos.byteIndex, 0)
	reader := bufio.NewReader(file)
	return &segmentReader{
		raw:       reader,
		curPosition:  pos.byteIndex,
		endPosition:  dataSize,
		checksumType: checksumType,
	}, nil
}*/
