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

package log

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetFileInformationByHandleEx = modkernel32.NewProc("GetFileInformationByHandleEx")
)

// isRemoved checks whether the file held by f is removed.
// On Windows isRemoved reads the DeletePending flags using the GetFileInformationByHandleEx.
// A file is not removed/unlinked as long as at least one process still own a
// file handle. A delete file is only marked as deleted, and file attributes
// can still be read. Only opening a file marked with 'DeletePending' will
// fail.
func isRemoved(f *os.File) bool {
	hdl := f.Fd()
	if hdl == uintptr(syscall.InvalidHandle) {
		return false
	}

	info := struct {
		AllocationSize int64
		EndOfFile      int64
		NumberOfLinks  int32
		DeletePending  bool
		Directory      bool
	}{}
	infoSz := unsafe.Sizeof(info)

	const class = 1 // FileStandardInfo
	r1, _, _ := syscall.Syscall6(
		procGetFileInformationByHandleEx.Addr(), 4, uintptr(hdl), class, uintptr(unsafe.Pointer(&info)), infoSz, 0, 0)
	if r1 == 0 {
		return true // assume file is removed if syscall errors
	}
	return info.DeletePending
}
