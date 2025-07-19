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

package filestream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/klauspost/compress/gzip"
)

const (
	magicHeader = "\x1f\x8b" // RFC 1952 magic bytes
)

type File interface {
	fs.File
	io.ReadSeekCloser

	// Name returns the name of the file as os.File would.
	Name() string
	// OSFile returns the underlying *os.File.
	OSFile() *os.File
	// IsGZIP returns true if the file is a GZIP file.
	IsGZIP() bool
}

// plainFile is a wrapper around an *os.File that implements the File interface.
// It acts as a proxy to the underlying *os.File.
type plainFile struct {
	*os.File
}

func (pf *plainFile) IsGZIP() bool {
	return false
}

func newPlainFile(f *os.File) *plainFile {
	return &plainFile{File: f}
}

// OSFile returns the underlying *os.File.
// This is part of the File interface.
func (pf *plainFile) OSFile() *os.File {
	return pf.File
}

type gzipSeekerReader struct {
	f        *os.File     // underlying gzip-compressed file
	gzr      *gzip.Reader // reader that yields uncompressed bytes
	buffSize int64        // buffer size used when emulating seeks

	// offset is the current offset in the *decompressed* stream. It's updated
	// by read.
	offset int64
}

func newGzipSeekerReader(f *os.File, buffSize int) (*gzipSeekerReader, error) {
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("could not create gzip reader: %w", err)
	}

	return &gzipSeekerReader{
		f:        f,
		gzr:      gzr,
		buffSize: int64(buffSize),
		offset:   0,
	}, nil
}

func (r *gzipSeekerReader) IsGZIP() bool {
	return true
}

// Stat returns Stat() of the underlying *os.File.
func (r *gzipSeekerReader) Stat() (fs.FileInfo, error) {
	return r.f.Stat()
}

// Name returns Name() of the underlying *os.File.
func (r *gzipSeekerReader) Name() string {
	return r.f.Name()
}

// OSFile returns the underlying *os.File.
func (r *gzipSeekerReader) OSFile() *os.File {
	return r.f
}

// Read reads plain data, decompressing it on the fly.
func (r *gzipSeekerReader) Read(p []byte) (n int, err error) {
	n, err = r.gzr.Read(p)

	r.offset += int64(n)
	return n, err
}

func (r *gzipSeekerReader) Close() error {
	gzerr := r.gzr.Close()
	if gzerr != nil {
		gzerr = fmt.Errorf("could not close gzip reader: %w", gzerr)
	}

	plainerr := r.f.Close()
	if plainerr != nil {
		plainerr = fmt.Errorf("could not close plain file: %w", plainerr)
	}

	return errors.Join(gzerr, plainerr)
}

// Seek seeks to offset within the *decompressed* data stream.
func (r *gzipSeekerReader) Seek(offset int64, whence int) (int64, error) {
	if whence >= io.SeekEnd {
		return 0, fmt.Errorf("gzipSeekerReader: SeekEnd (2) is unsupported")
	}

	finalOffset := offset

	// Convert SeekCurrent to absolute offset
	if whence == io.SeekCurrent {
		finalOffset += r.offset
	}

	if finalOffset < 0 {
		return 0, fmt.Errorf(
			"gzipSeekerReader: final offset must be non-negative, got: %d",
			finalOffset)
	}

	needsReset := (finalOffset < r.offset) || // move backwards
		(finalOffset == 0 && whence == io.SeekStart) // move to 0
	if needsReset {
		n, err := r.f.Seek(0, 0)
		if err != nil {
			return n, fmt.Errorf(
				"gzipSeekerReader: could not seek to 0: %w", err)
		}

		// it'll create a new reader, so this error can be safely ignored
		_ = r.gzr.Close()

		r.gzr, err = gzip.NewReader(r.f)
		if err != nil {
			return n, fmt.Errorf(
				"gzipSeekerReader: could not create new gzip reader: %w", err)
		}
		r.offset = 0

		// nothing to advance, we're done
		if finalOffset == 0 {
			return 0, nil
		}
	}

	// If we're already at the target offset, no need to advance
	if finalOffset == r.offset {
		return finalOffset, nil
	}

	// Calculate how many bytes we need to advance from current position
	bytesToAdvance := finalOffset - r.offset

	var err error
	if bytesToAdvance <= r.buffSize {
		_, err = r.Read(make([]byte, bytesToAdvance))
		if err != nil && !errors.Is(err, io.EOF) {
			return r.offset, fmt.Errorf(
				"gzipSeekerReader: could read bytesToAdvance=%d: %w",
				bytesToAdvance, err)
		}

		r.offset = finalOffset
		return r.offset, nil
	}

	chunks := bytesToAdvance / r.buffSize
	leftover := bytesToAdvance % r.buffSize
	buff := make([]byte, r.buffSize)
	for i := range chunks {
		_, err = r.gzr.Read(buff)
		if err != nil && !errors.Is(err, io.EOF) {
			return r.offset, fmt.Errorf(
				"gzipSeekerReader: could read chunk %d: %w", i, err)
		}
	}

	if leftover > 0 {
		_, err = r.Read(make([]byte, leftover))
		if err != nil && !errors.Is(err, io.EOF) {
			return r.offset, fmt.Errorf(
				"gzipSeekerReader: could read leftover %d: %w", leftover, err)
		}
	}

	r.offset = finalOffset
	return finalOffset, nil
}

// IsGZIP reports whether the file f starts with the GZIP magic header bytes as
// defined by RFC 1952. The file offset is reset to the original position before
// returning.
func IsGZIP(f *os.File) (bool, error) {
	// Remember current offset so we can reset it afterward.
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}
	// Ensure we always reset the offset.
	defer func() { _, _ = f.Seek(offset, io.SeekStart) }()

	// Read magic bytes, 2 bytes.
	header := make([]byte, len(magicHeader))
	if _, err := f.ReadAt(header, 0); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil // empty or too short file – definitely not gzip
		}
		return false, fmt.Errorf("GZIP: failed to read magic bytes: %w", err)
	}

	return bytes.Equal(header, []byte(magicHeader)), nil
}
