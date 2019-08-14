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

package osfs

import (
	"golang.org/x/sys/unix"

	"github.com/elastic/go-txfile/internal/vfs"
)

type syncState struct {
	noDataOnly bool
}

// Sync uses fsync or fdatasync (if vfs.SyncDataOnly flag is set).
//
// Handling write-back errors is at a mess in older linux kernels [1].
// With mixed read-write operations, there is a chance that write-back errors
// are never reported to user-space applications, as error flags are cleared in
// the caches.
// Error handling was somewhat improved in 4.13 [2][3], such that errors will
// actually be reported on fsync (more improvements have been added to 4.16).
//
// [1]: https://lwn.net/Articles/718734
// [2]: https://lwn.net/Articles/724307
// [3]: https://lwn.net/Articles/724232
func (f *File) Sync(flags vfs.SyncFlag) error {
	dataOnly := (flags & vfs.SyncDataOnly) != 0
	for {
		err := f.doSync(!f.state.sync.noDataOnly && dataOnly)
		if err == nil || (err != unix.EINTR && err != unix.EAGAIN) {
			return f.wrapErr("file/sync", err)
		}
	}
}

func (f *File) doSync(dataOnly bool) error {
	if dataOnly {
		err := normalizeSysError(unix.Fdatasync(int(f.File.Fd())))
		if err == unix.ENOSYS {
			f.state.sync.noDataOnly = true
			return f.File.Sync()
		}
	}
	return f.File.Sync()
}
