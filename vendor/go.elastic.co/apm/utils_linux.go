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

package apm // import "go.elastic.co/apm"

import (
	"bytes"
	"syscall"
	"unsafe"
)

func currentProcessTitle() (string, error) {
	// PR_GET_NAME (since Linux 2.6.11)
	// Return the name of the calling thread, in the buffer pointed to by
	// (char *) arg2.  The buffer should allow space for up to 16 bytes;
	// the returned string will be null-terminated.
	var buf [16]byte
	if _, _, errno := syscall.RawSyscall6(
		syscall.SYS_PRCTL, syscall.PR_GET_NAME,
		uintptr(unsafe.Pointer(&buf[0])),
		0, 0, 0, 0,
	); errno != 0 {
		return "", errno
	}
	return string(buf[:bytes.IndexByte(buf[:], 0)]), nil
}
