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
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
)

// Reader sets an upper limited on line length. Lines longer
// then the max configured line length will be snapped short.
type LimitReader struct {
	reader   reader.Reader
	maxBytes int
}

// New creates a new reader limiting the line length.
func NewLimitReader(r reader.Reader, maxBytes int) *LimitReader {
	return &LimitReader{reader: r, maxBytes: maxBytes}
}

// Next returns the next line.
func (r *LimitReader) Next() (reader.Message, error) {
	message, err := r.reader.Next()
	return r.limit(message, err)
}

// limit truncates the message content to maxBytes, flagging the message as
// truncated. It is shared by Next and FlushPending so a flushed message is
// limited exactly like any other.
func (r *LimitReader) limit(message reader.Message, err error) (reader.Message, error) {
	if len(message.Content) > r.maxBytes {
		tmp := make([]byte, r.maxBytes)
		n := copy(tmp, message.Content)
		if n != r.maxBytes {
			return message, fmt.Errorf("unexpected number of bytes were copied, %d instead of limit %d", n, r.maxBytes)
		}
		message.Content = tmp
		message.AddFlagsWithKey("log.flags", "truncated")
	}
	return message, err
}

var _ reader.BufferedFlusher = (*LimitReader)(nil)

// PendingDeadline implements reader.BufferedFlusher by delegating to the
// wrapped reader.
func (r *LimitReader) PendingDeadline() (bool, time.Time) {
	return reader.PendingDeadline(r.reader)
}

// FlushPending implements reader.BufferedFlusher. A message flushed from the
// wrapped reader is truncated to maxBytes like the ones returned by Next.
func (r *LimitReader) FlushPending(timedOut bool) (reader.Message, bool, error) {
	message, ok, err := reader.FlushPending(r.reader, timedOut)
	if !ok {
		return message, ok, err
	}
	message, err = r.limit(message, err)
	return message, ok, err
}

func (r *LimitReader) Close() error {
	return r.reader.Close()
}
