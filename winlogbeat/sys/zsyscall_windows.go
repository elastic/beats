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

package sys

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

var (
	modkernel = windows.NewLazySystemDLL("Kernel32.dll")

	procSystemTimeToFileTime = modkernel.NewProc("SystemTimeToFileTime")
)

func SystemTimeToFileTime(systemTime *windows.Systemtime, fileTime *windows.Filetime) error {
	r1, _, err := syscall.SyscallN(procSystemTimeToFileTime.Addr(), uintptr(unsafe.Pointer(systemTime)), uintptr(unsafe.Pointer(fileTime)))
	if r1 == 0 {
		return fmt.Errorf("error converting system time to file time: %w", err)
	}
	return nil
}
