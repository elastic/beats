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
	"runtime"
)

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

// trySyncPath provides a best-effort fsync on path (directory). The fsync is required by some
// filesystems, so to update the parents directory metadata to actually
// contain the new file being rotated in.
func trySyncPath(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // ignore error, sync on dir must not be necessarily supported by the FS
	}
	defer f.Close()
	//nolint:errcheck // ignore error
	f.Sync()
}

// pathEnsurePermissions checks if the file permissions for the given file match wantPerm.
// The permissions are updated using chmod if needed.
// No file will be created if the file does not yet exist.
func pathEnsurePermissions(path string, wantPerm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_RDWR, wantPerm)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	defer f.Close()
	return fileEnsurePermissions(f, wantPerm)
}

// fileEnsurePermissions checks if the file permissions for the given file
// matches wantPerm. If not fileEnsurePermissions tries to update
// the current permissions via chmod.
// The file is not created or updated if it does not exist.
func fileEnsurePermissions(f *os.File, wantPerm os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	fi, err := f.Stat()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	wantPerm = wantPerm & os.ModePerm
	perm := fi.Mode() & os.ModePerm
	if wantPerm == perm {
		return nil
	}

	return f.Chmod((fi.Mode() &^ os.ModePerm) | wantPerm)
}
