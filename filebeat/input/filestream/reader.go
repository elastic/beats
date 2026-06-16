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
	"errors"
	"io"

	"github.com/elastic/beats/v7/libbeat/reader"
)

// EOFLookaheadReader wraps a reader to provide a one-message lookahead buffer.
// It is designed to signal io.EOF on the message *before* the stream truly ends.
// It also takes an extra error to be considered EOF.
type EOFLookaheadReader struct {
	reader  reader.Reader
	nextMsg reader.Message
	nextErr error
	eofErr  error // error to consider EOF
}

// NewEOFLookaheadReader creates a new EOFLookaheadReader.
// If eofErr is not nil and r returns it, it's considered EOF.
// The call to Next will return the error and set Message.Private to io.EOF.
// It immediately reads the first message to prime the buffer.
func NewEOFLookaheadReader(r reader.Reader, eofErr error) *EOFLookaheadReader {
	if eofErr == nil {
		eofErr = io.EOF
	}
	lar := &EOFLookaheadReader{
		reader: r,
		eofErr: eofErr,
	}

	lar.nextMsg, lar.nextErr = lar.reader.Next()

	return lar
}

// Next returns the next message from the reader.
// When the underlying reader returns io.EOF, the current (and last) message
// is returned with its Private field set to io.EOF. If eofErr was set, when the
// underlying reader returns it, Next return the error and set Message.Private
// to io.EOF.
// Any subsequent call to Next will return a zero Message and io.EOF.
func (r *EOFLookaheadReader) Next() (reader.Message, error) {
	currentMsg := r.nextMsg
	currentErr := r.nextErr

	// lookahead
	r.nextMsg, r.nextErr = r.reader.Next()

	// If the *next* message fetch resulted in EOF, we need to signal it
	// on the *current* message that we are about to return.
	if errors.Is(r.nextErr, io.EOF) || errors.Is(currentErr, r.eofErr) {
		if !currentMsg.IsEmpty() {
			currentMsg.Private = io.EOF
		}
	}

	return currentMsg, currentErr
}

// Close closes the underlying reader.
func (r *EOFLookaheadReader) Close() error {
	return r.reader.Close()
}
