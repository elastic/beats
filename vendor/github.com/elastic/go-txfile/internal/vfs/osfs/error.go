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
	"os"

	"github.com/elastic/go-txfile/internal/vfs"
)

// errKind tries to find an appropriate error kind for an os error.
// If there is no checking predicate in the os package, sysErrKind is used to
// map OS specific error codes  to error kinds, such that a common and unified
// set of error codes will be available to users of vfs.
func errKind(err error) vfs.Kind {
	if os.IsPermission(err) {
		return vfs.ErrPermission
	}
	if os.IsExist(err) {
		return vfs.ErrExist
	}
	if os.IsNotExist(err) {
		return vfs.ErrExist
	}

	switch err {
	case os.ErrClosed:
		return vfs.ErrClosed
	default:
		return sysErrKind(err)
	}
}

// normalizeSysError returns the underlying error or nil, if the underlying
// error indicates it is no error.
func normalizeSysError(err error) error {
	err = underlyingError(err)
	if err == nil || err == errno0 {
		return nil
	}
	return err
}

func underlyingError(in error) error {
	switch err := in.(type) {
	case *os.PathError:
		return err.Err

	case *os.LinkError:
		return err.Err

	case *os.SyscallError:
		return err.Err

	default:
		return err
	}
}
