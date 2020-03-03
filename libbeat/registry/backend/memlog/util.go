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
	"hash"
	"hash/fnv"
	"io"
	"os"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/elastic/beats/v7/libbeat/registry/backend"
)

var errno0 = syscall.Errno(0)

type hashFn func(k backend.Key) uint64

type ensureWriter struct {
	w io.Writer
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

func newHash() hash.Hash64 {
	return fnv.New64a()
}

func newHashFn() hashFn {
	fn := newHash()
	return func(k backend.Key) uint64 {
		fn.Write(unsafeKeyRef(k))
		hash := fn.Sum64()
		fn.Reset()
		return hash
	}
}

func trySyncPath(path string) {
	// best-effort fsync on path (directory). The fsync is required by some
	// filesystems, so to update the parents directory metadata to actually
	// contain the new file being rotated in.
	f, err := os.Open(path)
	if err != nil {
		return // ignore error, sync on dir must not be necessarily supported by the FS
	}
	defer f.Close()
	syncFile(f)
}

func normalizeIOError(err error) error {
	if err == nil || err == errno0 {
		return nil
	}
	return err
}

func isIOError(err error) bool {
	return err == syscall.EIO ||
		// space/quota
		err == syscall.ENOSPC || err == syscall.EDQUOT || err == syscall.EFBIG ||
		// network
		err == syscall.ECONNRESET || err == syscall.ENETDOWN || err == syscall.ENETUNREACH
}

func isRetryErr(err error) bool {
	return err == syscall.EINTR || err == syscall.EAGAIN
}

func unsafeKeyRef(k backend.Key) []byte {
	str := (*reflect.StringHeader)(unsafe.Pointer(&k))
	hdr := reflect.SliceHeader{Data: str.Data, Len: str.Len, Cap: str.Len}
	return *(*[]byte)(unsafe.Pointer(&hdr))
}
