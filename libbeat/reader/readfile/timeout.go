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

package readfile

import (
	"errors"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
)

var (
	errTimeout = errors.New("timeout")
)

// TimeoutReader signals a configurable timeout error if no new line can be
// returned in time. It is fully synchronous — no goroutine, no channel handoff,
// no read-ahead — so it is safe over a reader that reuses the buffer backing
// Content.
//
// It enforces the timeout via a read deadline: the wrapped reader (or one it
// wraps) must implement reader.DeadlineSetter so the underlying blocking wait
// can be bounded. filestream's file reader, the log input's file reader,
// journald and kafka all honor deadlines. When the wrapped reader does not (a
// finite source such as awss3's object reads, which return on their own via EOF
// or the SDK request timeout), TimeoutReader reads directly: there is no
// never-returning read to bound, so no timeout is needed.
type TimeoutReader struct {
	reader  reader.Reader
	timeout time.Duration
	signal  error

	probed bool // whether deadline support has been determined
	sync   bool // wrapped reader honors deadlines -> timeout enforced via deadline
}

// NewTimeoutReader returns a new timeout reader from an input line reader.
func NewTimeoutReader(reader reader.Reader, signal error, t time.Duration) *TimeoutReader {
	if signal == nil {
		signal = errTimeout
	}

	return &TimeoutReader{
		reader:  reader,
		signal:  signal,
		timeout: t,
	}
}

// Next returns the next line. If no line was returned before the timeout (for a
// deadline-aware reader), the configured timeout error is returned.
func (r *TimeoutReader) Next() (reader.Message, error) {
	if !r.probed {
		r.probed = true
		// Determine once whether the wrapped reader (or one it wraps) honors
		// deadlines. The probe clears any deadline it might set.
		r.sync = reader.SetReadDeadline(r.reader, time.Time{})
	}

	if !r.sync {
		// Reader does not support deadlines: it returns on its own (finite source),
		// so no timeout is needed and no goroutine is spawned to bound it.
		return r.reader.Next()
	}

	reader.SetReadDeadline(r.reader, time.Now().Add(r.timeout))
	msg, err := r.reader.Next()
	reader.SetReadDeadline(r.reader, time.Time{})
	if errors.Is(err, reader.ErrReadDeadline) {
		return reader.Message{}, r.signal
	}
	return msg, err
}

func (r *TimeoutReader) Close() error {
	return r.reader.Close()
}
