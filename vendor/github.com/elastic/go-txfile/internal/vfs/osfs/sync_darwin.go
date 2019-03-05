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

type syncState struct{}

// Sync uses fnctl or fsync in order to flush the file buffers to disk.
// According to the darwin fsync man page[1], usage of sync is not safe. On
// darwin, fsync will only flush the OS file cache to disk, but this won't
// enforce a cache flush on the drive itself. Without forcing the cache flush,
// writes can still be out of order or get lost on power failure.
// According to the man page[1] fcntl with F_FULLFSYNC[2] is required. F_FULLFSYNC
// might not be supported for the current file system. In this case we will
// fallback to fsync.
//
// [1]: https://www.unix.com/man-page/osx/2/fsync
// [2]: https://www.unix.com/man-page/osx/2/fcntl
func (f *File) Sync(flags vfs.SyncFlag) error {
	err := f.doSync(flags)
	return f.wrapErr("file/sync", err)
}

func (f *File) doSync(flags vfs.SyncFlag) error {
	for {
		_, err := unix.FcntlInt(f.File.Fd(), unix.F_FULLFSYNC, 0)
		err = normalizeSysError(err)
		if err == nil || isIOError(err) {
			return err
		}

		if isRetryErr(err) {
			continue
		}

		// XXX: shall we 'guard' the second fsync via ENOTTY, EINVAL, ENXIO ?
		//      Question: What happens to the error status when calling fsync,
		//                if F_FULLFSYNC did actually fail due to an IO error, not
		//                captured by isIOError?
		err = f.File.Sync()
		if isRetryErr(err) {
			continue
		}
		return err
	}
}

func isIOError(err error) bool {
	return err == unix.EIO ||
		// space/quota
		err == unix.ENOSPC || err == unix.EDQUOT || err == unix.EFBIG ||
		// network
		err == unix.ECONNRESET || err == unix.ENETDOWN || err == unix.ENETUNREACH
}

func isRetryErr(err error) bool {
	return err == unix.EINTR || err == unix.EAGAIN
}
