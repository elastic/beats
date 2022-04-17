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
	"io"
	"time"

	"github.com/menderesk/beats/v7/libbeat/reader"
)

var (
	errTimeout = errors.New("timeout")
)

// TimeoutReader will signal some configurable timeout error if no
// new line can be returned in time.
type TimeoutReader struct {
	reader  reader.Reader
	timeout time.Duration
	signal  error
	running bool
	ch      chan lineMessage
	done    chan struct{}
}

type lineMessage struct {
	line reader.Message
	err  error
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
		ch:      make(chan lineMessage, 1),
		done:    make(chan struct{}),
	}
}

// Next returns the next line. If no line was returned before timeout, the
// configured timeout error is returned.
// For handline timeouts a goroutine is started for reading lines from
// configured line reader. Only when underlying reader returns an error, the
// goroutine will be finished.
func (r *TimeoutReader) Next() (reader.Message, error) {
	if !r.running {
		r.running = true
		go func() {
			for {
				message, err := r.reader.Next()
				select {
				case <-r.done:
					return
				case r.ch <- lineMessage{message, err}:
					if err != nil {
						return
					}
				}
			}
		}()
	}
	timer := time.NewTimer(r.timeout)
	select {
	case msg := <-r.ch:
		if msg.err != nil {
			r.running = false
		}
		timer.Stop()
		return msg.line, msg.err
	case <-timer.C:
		return reader.Message{}, r.signal
	case <-r.done:
		return reader.Message{}, io.EOF
	}
}

func (r *TimeoutReader) Close() error {
	close(r.done)

	return r.reader.Close()
}
