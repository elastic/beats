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

const _FileFlagNoBuffering = 0x20000000

// ReadOpen opens a file for reading only
// As Windows blocks deleting a file when its open, some special params are passed here.
func ReadOpen(path string, diskCacheOff bool) (*os.File, error) {
	// Set all write flags
	// This indirectly calls syscall_windows::Open method https://github.com/golang/go/blob/7ebcf5eac7047b1eef2443eda1786672b5c70f51/src/syscall/syscall_windows.go#L251
	// As FILE_SHARE_DELETE cannot be passed to Open, os.CreateFile must be implemented directly

	// This is mostly the code from syscall_windows::Open. Only difference is passing the Delete flag
	// TODO: Open pull request to Golang so also Delete flag can be set
	if len(path) == 0 {
		return nil, fmt.Errorf("File '%s' not found. Error: %v", path, syscall.ERROR_FILE_NOT_FOUND)
	}

	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("Error converting to UTF16: %v", err)
	}

	var access uint32
	access = syscall.GENERIC_READ

	sharemode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)

	var sa *syscall.SecurityAttributes

	var createmode uint32

	createmode = syscall.OPEN_EXISTING
	attrs := syscall.FILE_ATTRIBUTE_NORMAL
	if diskCacheOff {
		attrs = attrs | _FileFlagNoBuffering
	}

	handle, err := syscall.CreateFile(pathp, access, sharemode, sa, createmode, attrs, 0)

	if err != nil {
		return nil, fmt.Errorf("Error creating file '%s': %v", path, err)
	}

	return os.NewFile(uintptr(handle), path), nil
}
