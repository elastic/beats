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

package memlog

import (
	"io"
	"os"
	"syscall"
)

// ensureWriter writes the buffer to the underlying writer
// for as long as w returns a retryable error (e.g. EAGAIN)
// or the input buffer has been exhausted.
//
// XXX: this code was written and tested with go1.13 and go1.14, which does not
//      handled EINTR. Some users report EINTR getting triggered more often in
//      go1.14 due to changes in the signal handling for implementing
//      preemption.
//      In future versions EINTR will be handled by go for us.
//      See: https://github.com/golang/go/issues/38033
type ensureWriter struct {
	w io.Writer
}

// countWriter keeps track of the amount of bytes written over time.
type countWriter struct {
	n uint64
	w io.Writer
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += uint64(n)
	return n, err
}

func (e *ensureWriter) Write(p []byte) (int, error) {
	var N int
	for len(p) > 0 {
		n, err := e.w.Write(p)
		N, p = N+n, p[n:]
		if isRetryErr(err) {
			return N, err
		}
	}
	return N, nil
}

func isRetryErr(err error) bool {
	return err == syscall.EINTR || err == syscall.EAGAIN
}

// trySyncPath provides a best-effort fsync on path (directory). The fsync is required by some
// filesystems, so to update the parents directory metadata to actually
// contain the new file being rotated in.
func trySyncPath(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // ignore error, sync on dir must not be necessarily supported by the FS
	}
	defer f.Close()
	syncFile(f)
}
