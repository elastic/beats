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
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

type segmentManager struct {
	segments []segmentFile
}

// A wrapper around the file handle and metadata for a single segment file.
type segmentFile struct {
	id   segmentID
	size int64
	file *os.File
}

// Each data frame has 2 32-bit lengths and 1 32-bit checksum.
const frameMetadataSize = 12

// Sort order: we store loaded segments in ascending order by their id.
type bySegmentID []segmentFile

func (s bySegmentID) Len() int           { return len(s) }
func (s bySegmentID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s bySegmentID) Less(i, j int) bool { return s[i].size < s[j].size }

func segmentManagerForPath(path string) (*segmentManager, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read queue directory '%s': %w", path, err)
	}

	segments := []segmentFile{}
	for _, file := range files {
		components := strings.Split(file.Name(), ".")
		if len(components) == 2 && strings.ToLower(components[1]) == "seg" {
			// Parse the id as base-10 64-bit unsigned int. We ignore file names that
			// don't match the "[uint64].seg" pattern.
			if id, err := strconv.ParseUint(components[0], 10, 64); err == nil {
				segments = append(segments,
					segmentFile{id: segmentID(id), size: file.Size()})
			}
		}
	}
	sort.Sort(bySegmentID(segments))
	return &segmentManager{
		segments: segments,
	}, nil
}

type segmentReader struct {
	*bufio.Reader

	// The current byte offset of the reader within the file
	curPosition int64

	// The position at which this reader should stop reading. This is often
	// the end of the file, but it may be earlier when the queue is reading
	// and writing to the same segment.
	endPosition int64

	checksumType checksumType
}

type checksumType int

const (
	checksumTypeNone = iota
	checksumTypeCRC32
)

// A nil data frame with no error means this reader has no more frames.
func (reader *segmentReader) nextDataFrame() ([]byte, error) {
	if reader.curPosition >= reader.endPosition {
		return nil, nil
	}
	var frameLength uint32
	err := binary.Read(reader.Reader, binary.LittleEndian, &frameLength)
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
	_, err = io.ReadFull(reader.Reader, data)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't read data frame from disk: %w", err)
	}

	// Read the footer (length + checksum)
	var duplicateLength uint32
	err = binary.Read(reader.Reader, binary.LittleEndian, &duplicateLength)
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
	err = binary.Read(reader.Reader, binary.LittleEndian, &checksum)
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
