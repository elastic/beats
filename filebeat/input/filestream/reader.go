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
	reader reader.Reader
	eof    bool
	msg    reader.Message
	err    error
	eofErr error // error to consider EOF
}

// NewEOFLookaheadReader creates a new EOFLookaheadReader.
// If eofErr is not nil and r reruns it, it's considered EOF. The call to Next
// will return the error and set Message.Private to io.EOF.
// It immediately reads the first message to prime the buffer.
func NewEOFLookaheadReader(r reader.Reader, eofErr error) *EOFLookaheadReader {
	if eofErr == nil {
		eofErr = io.EOF
	}
	lar := &EOFLookaheadReader{
		reader: r,
		eofErr: io.EOF,
	}

	lar.msg, lar.err = lar.reader.Next()
	if errors.Is(lar.err, io.EOF) {
		lar.eof = true
	}

	return lar
}

// Next returns the next message from the reader.
// When the underlying reader returns io.EOF, the current (and last) message
// is returned with its Private field set to io.EOF. If eofErr was set, when the
// underlying reader returns it, Next return the error and set Message.Private
// to io.EOF.
// Any subsequent call to Next will return a zero Message and io.EOF.
func (r *EOFLookaheadReader) Next() (reader.Message, error) {
	if r.eof {
		return reader.Message{}, io.EOF
	}

	msgToReturn := r.msg
	errToReturn := r.err

	// lookahead
	r.msg, r.err = r.reader.Next()

	// If the *next* message fetch resulted in EOF, we need to signal it
	// on the *current* message that we are about to return.
	if errors.Is(r.err, io.EOF) || errors.Is(r.err, r.eofErr) {
		r.eof = true
		// what to do if private isn't nil?
		msgToReturn.Private = io.EOF
	}

	return msgToReturn, errToReturn
}

// Close closes the underlying reader.
func (r *EOFLookaheadReader) Close() error {
	return r.reader.Close()
}
