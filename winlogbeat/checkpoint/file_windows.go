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

package checkpoint

import (
	"os"
	"syscall"
)

const (
	_FILE_FLAG_WRITE_THROUGH = 0x80000000
)

func create(path string) (*os.File, error) {
	return createWriteThroughFile(path)
}

// createWriteThroughFile creates a file whose write operations do not go
// through any intermediary cache, they go directly to disk.
func createWriteThroughFile(path string) (*os.File, error) {
	if len(path) == 0 {
		return nil, syscall.ERROR_FILE_NOT_FOUND
	}
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	h, err := syscall.CreateFile(
		pathp, // Path
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,               // Access Mode
		uint32(syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE), // Share Mode
		nil,                   // Security Attributes
		syscall.CREATE_ALWAYS, // Create Mode
		uint32(syscall.FILE_ATTRIBUTE_NORMAL|_FILE_FLAG_WRITE_THROUGH), // Flags and Attributes
		0) // Template File

	return os.NewFile(uintptr(h), path), err
}
