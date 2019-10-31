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
	"syscall"

	"github.com/elastic/go-txfile/internal/vfs"
)

// File implements vfs.File for the current target operating system.
type File struct {
	*os.File
	state osFileState
}

type osFileState struct {
	mmap mmapState
	lock lockState
	sync syncState
}

var errno0 = syscall.Errno(0)

func Open(path string, mode os.FileMode) (*File, error) {
	flags := os.O_RDWR | os.O_CREATE
	f, err := os.OpenFile(path, flags, mode)
	if err != nil {
		return nil, vfs.Err("file/open", errKind(err), path, err)
	}
	return &File{File: f}, nil
}

func (f *File) Size() (int64, error) {
	stat, err := f.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

func (f *File) Stat() (os.FileInfo, error) {
	stat, err := f.File.Stat()
	return stat, f.wrapErr("file/stat", err)
}

func (f *File) Truncate(sz int64) error {
	err := f.File.Truncate(sz)
	return f.wrapErr("file/truncate", err)
}

func (f *File) wrapErr(op string, err error) error {
	return f.wrapErrKind(op, errKind(err), err)
}

func (f *File) wrapErrKind(op string, k vfs.Kind, err error) error {
	if err == nil {
		return nil
	}
	return vfs.Err(op, k, f.Name(), err)
}
