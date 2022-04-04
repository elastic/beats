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

//go:build !windows
// +build !windows

package file

import (
	"os"
	"strconv"
	"syscall"
)

type StateOS struct {
	Inode  uint64 `json:"inode," struct:"inode"`
	Device uint64 `json:"device," struct:"device"`
}

// GetOSState returns the FileStateOS for non windows systems
func GetOSState(info os.FileInfo) StateOS {
	stat := info.Sys().(*syscall.Stat_t)

	// Convert inode and dev to uint64 to be cross platform compatible
	fileState := StateOS{
		Inode:  uint64(stat.Ino),
		Device: uint64(stat.Dev),
	}

	return fileState
}

// IsSame file checks if the files are identical
func (fs StateOS) IsSame(state StateOS) bool {
	return fs.Inode == state.Inode && fs.Device == state.Device
}

func (fs StateOS) String() string {
	var buf [64]byte
	current := strconv.AppendUint(buf[:0], fs.Inode, 10)
	current = append(current, '-')
	current = strconv.AppendUint(current, fs.Device, 10)
	return string(current)
}

// ReadOpen opens a file for reading only
func ReadOpen(path string) (*os.File, error) {
	flag := os.O_RDONLY
	perm := os.FileMode(0)
	return os.OpenFile(path, flag, perm)
}

// IsRemoved checks wheter the file held by f is removed.
func IsRemoved(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		// if we got an error from a Stat call just assume we are removed
		return true
	}
	sysStat := stat.Sys().(*syscall.Stat_t)
	return sysStat.Nlink == 0
}

// InodeString returns the inode in string.
func (s *StateOS) InodeString() string {
	return strconv.FormatUint(s.Inode, 10)
}
