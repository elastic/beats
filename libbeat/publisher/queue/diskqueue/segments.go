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
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/logp"
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

	// If this segment was created during a previous session, the header
	// field will be populated during the initial scan on queue startup.
	// This is only used to support old schema versions, and is empty for
	// segments created in the current session.
	header *segmentHeader

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

	// The number of frames written to this segment during this session. This
	// is zero for any segment that was created in a previous session.
	// After a segment is done being written, this value is written to the
	// frameCount field in the segment file header.
	framesWritten uint32

	// The number of frames read from this segment during this session. This
	// does not necessarily equal the number of frames in the segment, even
	// after reading is complete, since the segment may have been partially
	// read during a previous session.
	//
	// Used to count how many frames still need to be acknowledged by consumers.
	framesRead uint64
}

type segmentHeader struct {
	// The schema version for this segment file. Current schema version is 1.
	version uint32

	// If the segment file has been completely written, this field contains
	// the number of data frames, which is used to track the number of
	// pending events left in the queue from previous sessions.
	// If the segment file has not been completely written, this field is zero.
	// Only present in schema version >= 1.
	frameCount uint32
}

const currentSegmentVersion = 1

// Segment headers are currently a 4-byte version plus a 4-byte frame count.
// In contexts where the segment may have been created by an earlier version,
// instead use (segmentHeader).sizeOnDisk() which accounts for the schema
// version of the target header.
const segmentHeaderSize = 8

// Sort order: we store loaded segments in ascending order by their id.
type bySegmentID []*queueSegment

func (s bySegmentID) Len() int           { return len(s) }
func (s bySegmentID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s bySegmentID) Less(i, j int) bool { return s[i].id < s[j].id }

func (header segmentHeader) sizeOnDisk() int {
	if header.version < 1 {
		// Schema 0 had nothing except the 4-byte version.
		return 4
	}
	// Current schema has a 4-byte version and 4-byte frame count.
	return 8
}

// Scan the given path for segment files, and return them in a list
// ordered by segment id.
func scanExistingSegments(logger *logp.Logger, pathStr string) ([]*queueSegment, error) {
	files, err := ioutil.ReadDir(pathStr)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read queue directory '%s': %w", pathStr, err)
	}

	segments := []*queueSegment{}
	for _, file := range files {
		/*if file.Size() <= segmentHeaderSize {
			// Ignore segments that don't have at least some data beyond the
			// header (this will always be true of segments we write unless there
			// is an error).
			continue
		}*/
		components := strings.Split(file.Name(), ".")
		if len(components) == 2 && strings.ToLower(components[1]) == "seg" {
			// Parse the id as base-10 64-bit unsigned int. We ignore file names that
			// don't match the "[uint64].seg" pattern.
			if id, err := strconv.ParseUint(components[0], 10, 64); err == nil {
				fullPath := path.Join(pathStr, file.Name())
				header, err := readSegmentHeaderWithFrameCount(fullPath)
				if header == nil {
					logger.Errorf("couldn't load segment file '%v': %v", fullPath, err)
					continue
				}
				// If we get an error but still got a valid header back, then we
				// were able to read at least some frames, so we keep this segment
				// but issue a warning.
				if err != nil {
					logger.Warnf(
						"error loading segment file '%v', data may be incomplete: %v",
						fullPath, err)
				}
				segments = append(segments, &queueSegment{
					id:        segmentID(id),
					header:    header,
					endOffset: segmentOffset(file.Size() - int64(header.sizeOnDisk())),
				})

				//newSegment, err := prescanSegment(logger, segmentID(id), fullPath)

				// If a segment is returned with an error, this means we were able to
				// read at least some data but the end of the file may be incomplete
				// or corrupted. In this case we add it to our list and read as much of
				// it as we can.
				/*if newSegment == nil {
					logger.Errorf("couldn't load segment file '%v': %v", fullPath, err)
				} else {
					if err != nil {
						logger.Warnf(
							"error loading segment file '%v', data may be incomplete: %v",
							fullPath, err)
					}
					segments = append(segments, newSegment)
				}*/
				/*frameCount, err := readFrameCount(path.Join(pathStr, file.Name()))
				if frameCount == 0 {
					logger.Errorf("")
				} else {
					if err != nil {
						// If there is an error but frameCount is still positive, it means
						// we
						logger.Warnf(
							"Error")
					}
					segments = append(segments, &queueSegment{
						id:            segmentID(id),
						endOffset:     segmentOffset(file.Size() - segmentHeaderSize),
						framesWritten: frameCount,
					})
				}*/
			}
		}
	}
	sort.Sort(bySegmentID(segments))
	return segments, nil
}

func (segment *queueSegment) sizeOnDisk() uint64 {
	var headerSize int
	if segment.header != nil {
		headerSize = segment.header.sizeOnDisk()
	} else {
		headerSize = segmentHeaderSize
	}
	return uint64(segment.endOffset) + uint64(headerSize)
}

// A helper function that returns the number of frames in an existing
// segment file, used during startup to count how many events are
// pending in the queue from a previous session.
// It first tries to read the frame count from the segment header
// (which requires segment schema version >= 1 and a successful prior
// shutdown). If this fails, it falls back to a manual scan through
// the file checking only the frame lengths.
func readFrameCount(path string) (uint32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf(
			"Couldn't open segment file '%s': %w", path, err)
	}
	defer file.Close()
	// Wrap the handle to retry non-fatal errors and always return the full
	// requested data length if possible.
	reader := autoRetryReader{file}
	header, err := readSegmentHeader(reader)
	if err != nil {
		return 0, err
	}
	if header.frameCount > 0 {
		return header.frameCount, nil
	}
	frameCount := uint32(0)
	for {
		var frameLength uint32
		err := binary.Read(reader, binary.LittleEndian, &frameLength)
		if err != nil {
			if err == io.EOF {
				// End of file at a frame boundary means we successfully scanned all
				// frames.
				return frameCount, nil
			}
			// All other errors are reported, but we still include the current
			// frameCount since we want to recover as many frames as we can.
			return frameCount, err
		}
		_, err = file.Seek(int64(frameLength), os.SEEK_CUR)
		if err != nil {
			// An error in seeking probably means an invalid length, which
			// indicates a truncated frame or data corruption, so return
			// without including it in our count.
			return frameCount, err
		}
		frameCount++
	}
}

// Should only be called from the reader loop. If successful, returns an open
// file handle positioned at the beginning of the segment's data region.
func (segment *queueSegment) getReader(
	queueSettings Settings,
) (*os.File, error) {
	path := queueSettings.segmentPath(segment.id)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Couldn't open segment %d: %w", segment.id, err)
	}
	// We don't need the header contents here, we just want to advance past the
	// header region, so discard the return value.
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
	err = writeSegmentHeader(file, 0)
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

// readSegmentHeaderWithFrameCount reads the header from the beginning
// of the file at the given path. If the header's frameCount is 0
// (whether because it is from an old version or because the segment
// file was not closed cleanly), it attempts to calculate it manually
// by scanning the file, and returns a struct with the "correct"
// frame count.
func readSegmentHeaderWithFrameCount(path string) (*segmentHeader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf(
			"couldn't open segment file '%s': %w", path, err)
	}
	defer file.Close()
	// Wrap the handle to retry non-fatal errors and always return the full
	// requested data length if possible, then read the raw header.
	reader := autoRetryReader{file}
	header, err := readSegmentHeader(reader)
	if err != nil {
		return nil, err
	}
	// If the header has a positive frame count then there is
	// no more work to do, so return immediately.
	if header.frameCount > 0 {
		return header, nil
	}
	// If we made it here, we loaded a valid header but the frame count is
	// zero, so we need to check it with a manual scan.
	for {
		var frameLength uint32
		err = binary.Read(reader, binary.LittleEndian, &frameLength)
		if err != nil {
			// EOF at a frame boundary means we successfully scanned all frames.
			if err == io.EOF && header.frameCount > 0 {
				return header, nil
			}
			// All other errors mean we are done scanning, exit the loop.
			break
		}
		// Try to advance to the next frame.
		_, err = file.Seek(int64(frameLength), os.SEEK_CUR)
		if err != nil {
			// An error in seeking probably means an invalid length, which
			// indicates a truncated frame or data corruption, so end the
			// loop without including it in our count.
			break
		}
		header.frameCount++
	}
	// If we ended up here instead of returning directly, then
	// we encountered an error. We still return a valid header as
	// long as we successfully scanned at least one frame first.
	if header.frameCount > 0 {
		return header, err
	}
	return nil, err
}

// readSegmentHeader decodes a raw header from the given reader and
// returns it as a struct.
func readSegmentHeader(in io.Reader) (*segmentHeader, error) {
	header := &segmentHeader{}
	err := binary.Read(in, binary.LittleEndian, &header.version)
	if err != nil {
		return nil, err
	}
	if header.version > currentSegmentVersion {
		return nil, fmt.Errorf("Unrecognized schema version %d", header.version)
	}
	if header.version >= 1 {
		err = binary.Read(in, binary.LittleEndian, &header.frameCount)
		if err != nil {
			return nil, err
		}
	}
	return header, nil
}

// writeSegmentHeader seeks to the beginning of the given file handle and
// writes a segment header with the current schema version, containing the
// given frameCount.
func writeSegmentHeader(out *os.File, frameCount uint32) error {
	_, err := out.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	version := uint32(currentSegmentVersion)
	err = binary.Write(out, binary.LittleEndian, version)
	if err != nil {
		return err
	}
	err = binary.Write(out, binary.LittleEndian, frameCount)
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
