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

import "github.com/elastic/go-txfile/internal/vfs"

type syncState struct{}

// Sync uses FlushFileBuffers to flush all file buffers to disk.
// For more information about the operation executed check the FlushFileBuffers API docs[1].
//
// Depending on Windows+driver versions or system wide settings,
// FlushFileBuffers might not be reliable. While FlushFileBuffers flushes the
// OS disk cache, we require a FLUSH_CACHE to be executed and honored by the
// driver and the device. Otherwise we might suffer data loss and file corruption.
// Also see [2] and [3].
//
// Enabling write caching on the disk [4] can disable the FLUSH_CACHE command,
// potentially leading to data loss and file corruption if the disk looses
// power.
//
// Check [5], for why we don't want to use write through.
//
// [1]: https://msdn.microsoft.com/de-de/library/windows/desktop/aa364439(v=vs.85).aspx
// [2]: https://blogs.msdn.microsoft.com/oldnewthing/20100909-00/?p=12913
// [3]: https://blogs.msdn.microsoft.com/oldnewthing/20170510-00/?p=95505/
// [4]: https://blogs.msdn.microsoft.com/emberger/2009/07/30/the-checkbox-that-saves-you-hours/
// [5]: https://perspectives.mvdirona.com/2008/04/disks-lies-and-damn-disks/
func (f *File) Sync(flags vfs.SyncFlag) error {
	err := f.File.Sync() // stdlib already uses FlushFileBuffes, yay
	return f.wrapErr("file/sync", err)
}
