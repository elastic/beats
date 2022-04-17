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

package debug

import (
	"bytes"
	"io"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

const (
	offsetStart        = 100
	offsetEnd          = 100
	defaultMinBuffer   = 16 * 1024
	defaultMaxFailures = 100
)

type state int

const (
	initial state = iota
	running
	stopped
)

// CheckFunc func receive a slice of bytes and returns true if it match the predicate.
type CheckFunc func(offset int64, buf []byte) bool

// Reader is a debug reader used to check the values of specific bytes from an io.Reader,
// Is is useful is you want to detect if you have received garbage from a network volume.
type Reader struct {
	log           *logp.Logger
	reader        io.ReadCloser
	buffer        bytes.Buffer
	minBufferSize int
	maxFailures   int
	failures      int
	predicate     CheckFunc
	state         state
	offset        int64
}

// NewReader returns a debug reader.
func NewReader(
	log *logp.Logger,
	reader io.ReadCloser,
	minBufferSize int,
	maxFailures int,
	predicate CheckFunc,
) (*Reader, error) {
	return &Reader{
		log:           log,
		minBufferSize: minBufferSize,
		reader:        reader,
		maxFailures:   maxFailures,
		predicate:     predicate,
	}, nil
}

// Read will proxy the read call to the original reader and will periodically checks the values of
// bytes in the buffer.
func (r *Reader) Read(p []byte) (int, error) {
	if r.state == stopped {
		return r.reader.Read(p)
	}

	if r.state == running && r.failures > r.maxFailures {
		// cleanup any remaining bytes in the buffer.
		if r.buffer.Len() > 0 {
			r.predicate(r.offset, r.buffer.Bytes())
		}
		r.buffer = bytes.Buffer{}
		r.log.Info("Stopping debug reader, max execution reached")
		r.state = stopped
		return r.reader.Read(p)
	}

	if r.state == initial {
		r.log.Infof(
			"Starting debug reader with a buffer size of %d and max failures of %d",
			r.minBufferSize,
			r.maxFailures,
		)
		r.state = running
	}

	n, err := r.reader.Read(p)

	if n != 0 {
		r.buffer.Write(p[:n])
		if r.buffer.Len() >= r.minBufferSize {
			if r.failures < r.maxFailures && r.predicate(r.offset, r.buffer.Bytes()) {
				r.failures++
			}
			r.buffer.Reset()
		}
		r.offset += int64(n)
	}
	return n, err
}

func (r *Reader) Close() error {
	return r.reader.Close()
}

func makeNullCheck(log *logp.Logger, minSize int) CheckFunc {
	// create a slice with null bytes to match on the buffer.
	pattern := make([]byte, minSize, minSize)
	return func(offset int64, buf []byte) bool {
		idx := bytes.Index(buf, pattern)
		if idx <= 0 {
			offset += int64(len(buf))
			return false
		}
		reportNull(log, offset+int64(idx), idx, buf)
		return true
	}
}

func reportNull(log *logp.Logger, offset int64, idx int, buf []byte) {
	relativePos, surround := summarizeBufferInfo(idx, buf)
	log.Debugf(
		"Matching null byte found at offset %d (position %d in surrounding string: %s, bytes: %+v",
		offset,
		relativePos,
		string(surround),
		surround)
}

func summarizeBufferInfo(idx int, buf []byte) (int, []byte) {
	startAt := idx - offsetStart
	var relativePos int
	if startAt < 0 {
		startAt = 0
		relativePos = idx
	} else {
		relativePos = offsetStart
	}

	endAt := idx + offsetEnd
	if endAt >= len(buf) {
		endAt = len(buf)
	}
	surround := buf[startAt:endAt]
	return relativePos, surround
}

// AppendReaders look into the current enabled log selector and will add any debug reader that match
// the selectors.
func AppendReaders(reader io.ReadCloser) (io.ReadCloser, error) {
	var err error

	if logp.HasSelector("detect_null_bytes") || logp.HasSelector("*") {
		log := logp.NewLogger("detect_null_bytes")
		if reader, err = NewReader(
			log,
			reader,
			defaultMinBuffer,
			defaultMaxFailures,
			makeNullCheck(log, 4),
		); err != nil {
			return nil, err
		}
	}
	return reader, nil
}
