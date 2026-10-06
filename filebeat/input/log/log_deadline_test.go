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

package log

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
)

// deadlineMockSource returns its data once and then reports EOF while remaining
// continuable, mimicking a file being tailed (no more data yet, but the source
// stays open).
type deadlineMockSource struct {
	data []byte
	pos  int
}

func (s *deadlineMockSource) Read(p []byte) (int, error) {
	if s.pos < len(s.data) {
		n := copy(p, s.data[s.pos:])
		s.pos += n
		return n, nil
	}
	return 0, io.EOF
}

func (s *deadlineMockSource) Close() error      { return nil }
func (s *deadlineMockSource) Name() string      { return "mock" }
func (s *deadlineMockSource) Removed() bool     { return false }
func (s *deadlineMockSource) Continuable() bool { return true }
func (s *deadlineMockSource) HasState() bool    { return true }
func (s *deadlineMockSource) Stat() (os.FileInfo, error) {
	return deadlineMockInfo{size: int64(len(s.data))}, nil
}

type deadlineMockInfo struct {
	os.FileInfo
	size int64
}

func (i deadlineMockInfo) Size() int64 { return i.size }

// TestLogReadDeadline verifies the log input's source honors a read deadline.
// The multiline timeout is enforced synchronously (no goroutine) by setting a
// deadline before each read; while tailing at EOF the source must return
// reader.ErrReadDeadline when it elapses instead of blocking in backoff
// forever. Without this, the multiline timeout reader could never flush a
// pending event at EOF, dropping the last multiline event.
func TestLogReadDeadline(t *testing.T) {
	src := &deadlineMockSource{data: []byte("a line\n")}
	lf, err := NewLog(logp.NewNopLogger(), src, LogConfig{
		Backoff:       time.Millisecond,
		BackoffFactor: 2,
		MaxBackoff:    10 * time.Millisecond,
		CloseInactive: time.Hour,
	})
	require.NoError(t, err)

	// A generous deadline lets the data read complete.
	buf := make([]byte, 64)
	lf.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := lf.Read(buf)
	require.NoError(t, err)
	require.Positive(t, n)

	// At EOF with a short deadline, Read must return ErrReadDeadline rather than
	// blocking in backoff.
	lf.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	start := time.Now()
	_, err = lf.Read(buf)
	require.ErrorIs(t, err, reader.ErrReadDeadline)
	require.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond,
		"Read returned before the deadline elapsed")
}
