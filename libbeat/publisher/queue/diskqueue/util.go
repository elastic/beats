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

package diskqueue

import (
	"errors"
	"io"
	"syscall"
)

// A wrapper for an io.Reader that tries to read the full number of bytes
// requested, retrying on EAGAIN and EINTR, and returns an error if
// and only if the number of bytes read is less than requested.
// This is similar to io.ReadFull but with retrying.
type autoRetryReader struct {
	wrapped io.Reader
}

func (r autoRetryReader) Read(p []byte) (int, error) {
	bytesRead := 0
	reader := r.wrapped
	n, err := reader.Read(p)
	for n < len(p) {
		if err != nil && !readErrorIsRetriable(err) {
			return bytesRead + n, err
		}
		// If there is an error, it is retriable, so advance p and try again.
		bytesRead += n
		p = p[n:]
		n, err = reader.Read(p)
	}
	return bytesRead + n, nil
}

func readErrorIsRetriable(err error) bool {
	return errors.Is(err, syscall.EINTR) || errors.Is(err, syscall.EAGAIN)
}

// writeErrorIsRetriable returns true if the given IO error can be
// immediately retried.
func writeErrorIsRetriable(err error) bool {
	return errors.Is(err, syscall.EINTR) || errors.Is(err, syscall.EAGAIN)
}

// callbackRetryWriter is an io.Writer that wraps another writer and enables
// write-with-retry. When a Write encounters an error, it is passed to the
// retry callback. If the callback returns true, the the writer retries
// any unwritten portion of the input, otherwise it passes the error back
// to the caller.
// This helper is specifically for working with the writer loop, which needs
// to be able to retry forever at configurable intervals, but also cancel
// immediately if the queue is closed.
// This writer is unbuffered. In particular, it is safe to modify the
// "wrapped" field in-place as long as it isn't captured by the callback.
type callbackRetryWriter struct {
	wrapped io.Writer

	// The retry callback is called with the error that was produced and whether
	// this is the first (subsequent) error arising from this particular
	// write call.
	retry func(err error, firstTime bool) bool
}

func (w callbackRetryWriter) Write(p []byte) (int, error) {
	// firstTime tracks whether the current error is the first subsequent error
	// being passed to the retry callback. This is so that the callback can
	// reset its internal counters in case it is using exponential backoff or
	// a retry limit.
	firstTime := true
	bytesWritten := 0
	writer := w.wrapped
	n, err := writer.Write(p)
	for n < len(p) {
		if err != nil {
			shouldRetry := w.retry(err, firstTime)
			firstTime = false
			if !shouldRetry {
				return bytesWritten + n, err
			}
		} else {
			// If we made progress without an error, reset firstTime.
			firstTime = true
		}
		// Advance p and try again.
		bytesWritten += n
		p = p[n:]
		n, err = writer.Write(p)
	}
	return bytesWritten + n, nil
}
