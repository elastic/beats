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

package file

import (
	"errors"
	"os"
)

// A FileInfo describes a file and is returned by Stat and Lstat.
type FileInfo interface {
	os.FileInfo
	UID() (int, error) // UID of the file owner. Returns an error on non-POSIX file systems.
	GID() (int, error) // GID of the file owner. Returns an error on non-POSIX file systems.
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func Stat(name string) (FileInfo, error) {
	return stat(name, os.Stat)
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func Lstat(name string) (FileInfo, error) {
	return stat(name, os.Lstat)
}

// Wrap wraps the given os.FileInfo and returns a FileInfo in order to expose
// the UID and GID in a uniform manner across operating systems.
func Wrap(info os.FileInfo) (FileInfo, error) {
	return wrap(info)
}

type fileInfo struct {
	os.FileInfo
	uid *int
	gid *int
}

func (f fileInfo) UID() (int, error) {
	if f.uid == nil {
		return -1, errors.New("uid not implemented")
	}

	return *f.uid, nil
}

func (f fileInfo) GID() (int, error) {
	if f.gid == nil {
		return -1, errors.New("gid not implemented")
	}

	return *f.gid, nil
}
