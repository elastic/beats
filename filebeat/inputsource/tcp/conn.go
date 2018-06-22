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

package tcp

import (
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
)

// ErrMaxReadBuffer returns when too many bytes was read on the io.Reader
var ErrMaxReadBuffer = errors.New("max read buffer reached")

// ResetableLimitedReader is based on LimitedReader but allow to reset the byte read and return a specific
// error when we reach the limit.
type ResetableLimitedReader struct {
	reader        io.Reader
	maxReadBuffer uint64
	byteRead      uint64
}

// NewResetableLimitedReader returns a new ResetableLimitedReader
func NewResetableLimitedReader(reader io.Reader, maxReadBuffer uint64) *ResetableLimitedReader {
	return &ResetableLimitedReader{
		reader:        reader,
		maxReadBuffer: maxReadBuffer,
	}
}

// Read reads the specified amount of byte
func (m *ResetableLimitedReader) Read(p []byte) (n int, err error) {
	if m.byteRead >= m.maxReadBuffer {
		return 0, ErrMaxReadBuffer
	}
	n, err = m.reader.Read(p)
	m.byteRead += uint64(n)
	return
}

// Reset resets the number of byte read
func (m *ResetableLimitedReader) Reset() {
	m.byteRead = 0
}

// IsMaxReadBufferErr returns true when the error is ErrMaxReadBuffer
func IsMaxReadBufferErr(err error) bool {
	return err == ErrMaxReadBuffer
}

// DeadlineReader allow read to a io.Reader to timeout, the timeout is refreshed on every read.
type DeadlineReader struct {
	conn    net.Conn
	timeout time.Duration
}

// NewDeadlineReader returns a new DeadlineReader
func NewDeadlineReader(c net.Conn, timeout time.Duration) *DeadlineReader {
	return &DeadlineReader{
		conn:    c,
		timeout: timeout,
	}
}

// Read reads the number of bytes from the reader
func (d *DeadlineReader) Read(p []byte) (n int, err error) {
	d.refresh()
	return d.conn.Read(p)
}

func (d *DeadlineReader) refresh() {
	d.conn.SetDeadline(time.Now().Add(d.timeout))
}
