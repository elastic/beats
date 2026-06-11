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

//go:build !integration

package readfile

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
)

func msg(s string) reader.Message {
	return reader.Message{Content: []byte(s), Bytes: len(s)}
}

// deadlineSource is a reader that honors read deadlines, mimicking the file
// reader: it returns queued messages, then on an empty queue blocks until the
// deadline elapses (returning reader.ErrReadDeadline) or it is closed. It
// implements reader.DeadlineSetter, so TimeoutReader uses its synchronous path.
type deadlineSource struct {
	mu       sync.Mutex
	msgs     []reader.Message
	idx      int
	reads    int
	deadline time.Time
	closed   chan struct{}
}

func newDeadlineSource(msgs ...reader.Message) *deadlineSource {
	return &deadlineSource{msgs: msgs, closed: make(chan struct{})}
}

func (s *deadlineSource) Next() (reader.Message, error) {
	s.mu.Lock()
	s.reads++
	if s.idx < len(s.msgs) {
		m := s.msgs[s.idx]
		s.idx++
		s.mu.Unlock()
		return m, nil
	}
	deadline := s.deadline
	s.mu.Unlock()

	var dl <-chan time.Time
	if !deadline.IsZero() {
		t := time.NewTimer(time.Until(deadline))
		defer t.Stop()
		dl = t.C
	}
	select {
	case <-s.closed:
		return reader.Message{}, io.EOF
	case <-dl:
		return reader.Message{}, reader.ErrReadDeadline
	}
}

func (s *deadlineSource) SetReadDeadline(t time.Time) bool {
	s.mu.Lock()
	s.deadline = t
	s.mu.Unlock()
	return true
}

func (s *deadlineSource) Close() error { close(s.closed); return nil }

func (s *deadlineSource) readCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads
}

// noDeadline does not implement reader.DeadlineSetter, so TimeoutReader reads it
// directly without enforcing a timeout. It is finite (returns io.EOF once
// drained), mimicking a source like awss3 that returns on its own.
type noDeadline struct {
	msgs []reader.Message
	idx  int
}

func newNoDeadline(msgs ...reader.Message) *noDeadline {
	return &noDeadline{msgs: msgs}
}

func (s *noDeadline) Next() (reader.Message, error) {
	if s.idx < len(s.msgs) {
		m := s.msgs[s.idx]
		s.idx++
		return m, nil
	}
	return reader.Message{}, io.EOF
}

func (s *noDeadline) Close() error { return nil }

// --- synchronous (deadline-aware) path ---

func TestTimeoutReaderSyncReturnsLines(t *testing.T) {
	src := newDeadlineSource(msg("a"), msg("b"), msg("c"))
	r := NewTimeoutReader(src, errors.New("sig"), time.Second)
	defer r.Close()

	for _, want := range []string{"a", "b", "c"} {
		m, err := r.Next()
		require.NoError(t, err)
		require.Equal(t, want, string(m.Content))
	}
}

func TestTimeoutReaderSyncDoesNotReadAhead(t *testing.T) {
	src := newDeadlineSource(msg("1"), msg("2"), msg("3"))
	r := NewTimeoutReader(src, errors.New("sig"), time.Second)
	defer r.Close()

	m, err := r.Next()
	require.NoError(t, err)
	require.Equal(t, "1", string(m.Content))
	// The synchronous path reads exactly one line per Next() — no read-ahead.
	require.Equal(t, 1, src.readCount())

	m, err = r.Next()
	require.NoError(t, err)
	require.Equal(t, "2", string(m.Content))
	require.Equal(t, 2, src.readCount())
}

func TestTimeoutReaderSyncSignalsTimeout(t *testing.T) {
	sig := errors.New("multiline timeout")
	src := newDeadlineSource() // no data -> blocks until the deadline
	r := NewTimeoutReader(src, sig, 20*time.Millisecond)
	defer r.Close()

	_, err := r.Next()
	require.ErrorIs(t, err, sig, "expected the timeout signal when no line arrives within the deadline")
}

// --- direct path (reader without deadline support, e.g. awss3) ---

func TestTimeoutReaderDirectReturnsLines(t *testing.T) {
	src := newNoDeadline(msg("a"), msg("b"))
	r := NewTimeoutReader(src, errors.New("sig"), time.Second)
	defer r.Close()

	for _, want := range []string{"a", "b"} {
		m, err := r.Next()
		require.NoError(t, err)
		require.Equal(t, want, string(m.Content))
	}
	_, err := r.Next()
	require.ErrorIs(t, err, io.EOF)
}
