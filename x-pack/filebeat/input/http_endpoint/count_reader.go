// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"errors"
	"io"
	"sync/atomic"
)

// errMaxInFlightExceeded is returned when the maximum in-flight bytes limit is exceeded.
var errMaxInFlightExceeded = errors.New("max_in_flight_bytes exceeded")

// countReader wraps an io.ReadCloser and atomically tracks the number of bytes
// read against a shared in-flight counter. When the hard limit (max) is exceeded,
// it returns ErrMaxInFlightExceeded.
type countReader struct {
	r        io.ReadCloser
	inFlight *atomic.Int64 // shared counter across all requests
	read     int64         // bytes read by this reader
	max      int64         // hard limit; 0 means no limit
	closed   bool          // track if Close has been called
}

// newCountReader creates a new countReader that wraps the given reader.
// The inFlight counter is shared across all requests and is incremented as bytes
// are read. If max is non-zero and inFlight exceeds max, ErrMaxInFlightExceeded
// is returned from Read.
func newCountReader(r io.ReadCloser, inFlight *atomic.Int64, max int64) *countReader {
	return &countReader{
		r:        r,
		inFlight: inFlight,
		max:      max,
	}
}

// Read reads from the underlying reader and updates the in-flight byte counter.
// If the max limit is exceeded, it returns ErrMaxInFlightExceeded.
func (m *countReader) Read(p []byte) (int, error) {
	n, err := m.r.Read(p)
	if n != 0 {
		m.read += int64(n)
		inFlight := m.inFlight.Add(int64(n))
		// Note: The check against max is subject to a benign race with concurrent
		// readers. The actual in-flight total may differ slightly from inFlight
		// by the time the check executes. This is acceptable as the limit is
		// intended to prevent memory exhaustion, not provide exact accounting.
		if m.max != 0 && inFlight > m.max {
			return n, errMaxInFlightExceeded
		}
	}
	return n, err
}

// Close closes the underlying reader and subtracts the bytes read by this reader
// from the shared in-flight counter.
func (m *countReader) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	m.inFlight.Add(-m.read)
	return m.r.Close()
}
