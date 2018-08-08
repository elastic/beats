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

// +build !windows

package osfs

import (
	"syscall"

	"github.com/elastic/go-txfile/internal/vfs"
)

// sysErrKind maps POSIX error codes to vfs related error codes.
func sysErrKind(err error) vfs.Kind {
	err = underlyingError(err)
	switch err {
	case syscall.EDQUOT, syscall.ENOSPC, syscall.ENFILE:
		return vfs.ErrNoSpace

	case syscall.EMFILE:
		return vfs.ErrFDLimit

	case syscall.ENOTDIR:
		return vfs.ErrResolvePath

	case syscall.ENOTSUP:
		return vfs.ErrNotSupported

	case syscall.EIO:
		return vfs.ErrIO

	case syscall.EDEADLK:
		return vfs.ErrLockFailed
	}

	return vfs.ErrOSOther
}
