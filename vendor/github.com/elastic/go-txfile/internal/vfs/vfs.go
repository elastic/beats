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

package vfs

import (
	"io"
)

type File interface {
	io.Closer
	io.WriterAt
	io.ReaderAt

	Name() string
	Size() (int64, error)
	Truncate(int64) error

	Lock(exclusive, blocking bool) error
	Unlock() error

	MMap(sz int) ([]byte, error)
	MUnmap([]byte) error

	// If a write/flush fails due to IO errors or the disk running out of space,
	// the kernel internally marks the error on the 'page'. Fsync will finally
	// return the error, but reset the error on failed writes. Subsequent fsync
	// operations will not report errors for former failed pages, even if the
	// pages are not written again. Therefore, if fsync fails, we must assume all
	// write operations - since the last successfull fsync - have failed and
	// reinitiate all writes.
	// According to [1] Linux, OpenBSD, and NetBSD are known to silently clear
	// errors on fsync fail.
	//
	// [1]: https://lwn.net/Articles/752098/
	Sync(flags SyncFlag) error
}

type SyncFlag uint8

const (
	SyncAll SyncFlag = 0

	// SyncDataOnly will only flush the file data, without enforcing an update on
	// the file metadata (like file size or modification time).
	SyncDataOnly SyncFlag = 1 << iota
)
