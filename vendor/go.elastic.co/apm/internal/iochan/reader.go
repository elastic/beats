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

package iochan

import (
	"sync"
)

// Reader is a channel-based io.Reader.
//
// Reader is safe for use in a single producer, single consumer pattern.
type Reader struct {
	// C can be used for receiving read requests.
	//
	// Once a read request is received, it must be responded
	// to, in order to avoid blocking the reader.
	C    <-chan ReadRequest
	c    chan ReadRequest
	resp chan readResponse

	mu          sync.Mutex
	readClosed  bool
	writeClosed bool
	readErr     error
}

// NewReader returns a new Reader.
func NewReader() *Reader {
	c := make(chan ReadRequest)
	return &Reader{
		C:    c,
		c:    c,
		resp: make(chan readResponse, 1),
	}
}

// CloseWrite closes reader.C. CloseWrite is idempotent,
// but must not be called concurrently with Read.
func (r *Reader) CloseWrite() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.writeClosed {
		r.writeClosed = true
		close(r.c)
	}
}

// CloseRead closes the reader such that any waiting or future
// Reads return err. Additional calls to CloseRead have no
// effect. CloseRead must not be called concurrently with
// ReadRequest.Respond.
func (r *Reader) CloseRead(err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.readClosed {
		r.readClosed = true
		r.readErr = err
		close(r.resp)
	}
	return nil
}

// Read sends a ReadRequest to r.C containing buf, and returns the
// response sent by the channel consumer via the read request's
// Response method.
func (r *Reader) Read(buf []byte) (int, error) {
	select {
	case <-r.resp:
		return 0, r.readErr
	case r.c <- ReadRequest{Buf: buf, response: r.resp}:
	}
	resp, ok := <-r.resp
	if !ok {
		return 0, r.readErr
	}
	return resp.N, resp.Err
}

// ReadRequest holds the buffer and response channel for a read request.
type ReadRequest struct {
	// Buf is the read buffer into which data should be read.
	Buf      []byte
	response chan<- readResponse
}

// Respond responds to the Read request. Respond must not be called
// concurrently with Reader.Close.
func (rr *ReadRequest) Respond(n int, err error) {
	rr.response <- readResponse{N: n, Err: err}
}

type readResponse struct {
	N   int
	Err error
}
