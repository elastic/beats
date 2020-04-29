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

// +build windows

package windows

import (
	"fmt"
	"unsafe"

	"github.com/pkg/errors"
)

// Syscalls
//sys   _GetFileVersionInfo(filename string, reserved uint32, dataLen uint32, data *byte) (success bool, err error) [!success] = version.GetFileVersionInfoW
//sys   _GetFileVersionInfoSize(filename string, handle uintptr) (size uint32, err error) = version.GetFileVersionInfoSizeW
//sys   _VerQueryValueW(data *byte, subBlock string, pBuffer *uintptr, len *uint32) (success bool, err error) [!success] = version.VerQueryValueW

// FixedFileInfo contains version information for a file. This information is
// language and code page independent. This is an equivalent representation of
// VS_FIXEDFILEINFO.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms646997(v=vs.85).aspx
type FixedFileInfo struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

// ProductVersion returns the ProductVersion value in string format.
func (info FixedFileInfo) ProductVersion() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(info.ProductVersionMS >> 16),
		(info.ProductVersionMS & 0xFFFF),
		(info.ProductVersionLS >> 16),
		(info.ProductVersionLS & 0xFFFF))
}

// FileVersion returns the FileVersion value in string format.
func (info FixedFileInfo) FileVersion() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(info.FileVersionMS >> 16),
		(info.FileVersionMS & 0xFFFF),
		(info.FileVersionLS >> 16),
		(info.FileVersionLS & 0xFFFF))
}

// VersionData is a buffer holding the data returned by GetFileVersionInfo.
type VersionData []byte

// QueryValue uses VerQueryValue to query version information from the a
// version-information resource. It returns responses using the first language
// and code point found in the resource. The accepted keys are listed in
// the VerQueryValue documentation (e.g. ProductVersion, FileVersion, etc.).
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms647464(v=vs.85).aspx
func (d VersionData) QueryValue(key string) (string, error) {
	type LangAndCodePage struct {
		Language uint16
		CodePage uint16
	}

	var dataPtr uintptr
	var size uint32
	if _, err := _VerQueryValueW(&d[0], `\VarFileInfo\Translation`, &dataPtr, &size); err != nil || size == 0 {
		return "", errors.Wrap(err, "failed to get list of languages")
	}

	offset := int(dataPtr - (uintptr)(unsafe.Pointer(&d[0])))
	if offset <= 0 || offset > len(d)-1 {
		return "", errors.New("invalid address")
	}

	l := *(*LangAndCodePage)(unsafe.Pointer(&d[offset]))

	subBlock := fmt.Sprintf(`\StringFileInfo\%04x%04x\%v`, l.Language, l.CodePage, key)
	if _, err := _VerQueryValueW(&d[0], subBlock, &dataPtr, &size); err != nil || size == 0 {
		return "", errors.Wrapf(err, "failed to query %v", subBlock)
	}

	offset = int(dataPtr - (uintptr)(unsafe.Pointer(&d[0])))
	if offset <= 0 || offset > len(d)-1 {
		return "", errors.New("invalid address")
	}

	str, _, err := UTF16BytesToString(d[offset : offset+int(size)*2])
	if err != nil {
		return "", errors.Wrap(err, "failed to decode UTF16 data")
	}

	return str, nil
}

// FixedFileInfo returns the fixed version information from a
// version-information resource. It queries the root block to get the
// VS_FIXEDFILEINFO value.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms647464(v=vs.85).aspx
func (d VersionData) FixedFileInfo() (*FixedFileInfo, error) {
	if len(d) == 0 {
		return nil, errors.New("use GetFileVersionInfo to initialize VersionData")
	}

	var dataPtr uintptr
	var size uint32
	if _, err := _VerQueryValueW(&d[0], `\`, &dataPtr, &size); err != nil {
		return nil, errors.Wrap(err, "VerQueryValue failed for \\")
	}

	offset := int(dataPtr - (uintptr)(unsafe.Pointer(&d[0])))
	if offset <= 0 || offset > len(d)-1 {
		return nil, errors.New("invalid address")
	}

	// Make a copy of the struct.
	ffi := *(*FixedFileInfo)(unsafe.Pointer(&d[offset]))

	return &ffi, nil
}

// GetFileVersionInfo retrieves version information for the specified file.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms647003(v=vs.85).aspx
func GetFileVersionInfo(filename string) (VersionData, error) {
	size, err := _GetFileVersionInfoSize(filename, 0)
	if err != nil {
		return nil, errors.Wrap(err, "GetFileVersionInfoSize failed")
	}

	data := make(VersionData, size)
	_, err = _GetFileVersionInfo(filename, 0, uint32(len(data)), &data[0])
	if err != nil {
		return nil, errors.Wrap(err, "GetFileVersionInfo failed")
	}

	return data, nil
}
