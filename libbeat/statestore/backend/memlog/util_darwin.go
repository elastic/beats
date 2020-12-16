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
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

var errno0 = syscall.Errno(0)

// syncFile implements the fsync operation for darwin. On darwin fsync is not
// reliable, instead the fcntl syscall with F_FULLFSYNC must be used.
func syncFile(f *os.File) error {
	for {
		_, err := unix.FcntlInt(f.Fd(), unix.F_FULLFSYNC, 0)
		rootCause := errorRootCause(err)
		if err == nil || isIOError(rootCause) {
			return err
		}

		if isRetryErr(err) {
			continue
		}

		err = f.Sync()
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

// normalizeSysError returns the underlying error or nil, if the underlying
// error indicates it is no error.
func errorRootCause(err error) error {
	for err != nil {
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = u.Unwrap()
	}

	if err == nil || err == errno0 {
		return nil
	}
	return err
}
