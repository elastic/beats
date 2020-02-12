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
	"syscall"

	"github.com/elastic/go-txfile/internal/vfs"
)

const (
	ERROR_DISK_FULL             syscall.Errno = 62
	ERROR_DISK_QUOTA_EXCEEDED   syscall.Errno = 1295
	ERROR_TOO_MANY_OPEN_FILES   syscall.Errno = 4
	ERROR_LOCK_FAILED           syscall.Errno = 167
	ERROR_CANT_RESOLVE_FILENAME syscall.Errno = 1921
)

// sysErrKind maps Windows error codes to vfs related error codes.
func sysErrKind(err error) vfs.Kind {
	switch underlyingError(err) {

	case ERROR_DISK_FULL, ERROR_DISK_QUOTA_EXCEEDED:
		return vfs.ErrNoSpace

	case ERROR_TOO_MANY_OPEN_FILES:
		return vfs.ErrFDLimit

	case ERROR_LOCK_FAILED:
		return vfs.ErrLockFailed

	case ERROR_CANT_RESOLVE_FILENAME:
		return vfs.ErrResolvePath

	default:
		return vfs.ErrOSOther
	}
}
