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

//go:build windows

package file_integrity

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	statusSuccess        uintptr = 0x00000000
	statusBufferTooSmall uintptr = 0xC0000023
	statusNoEAsOnFile    uintptr = 0xC0000225

	initialEABufferSize = 4096
	maxEABufferSize     = 65536 // Max size (64KB) for extended attributes on a file.
)

// ioStatusBlock is used by NT system calls to return status information.
// The Information field's meaning depends on the call; for NtQueryEaFile,
// it can return the required buffer size.
type ioStatusBlock struct {
	Status      uintptr
	Information uintptr
}

// fileFullEaInformation maps to the Windows FILE_FULL_EA_INFORMATION struct.
// It represents the header for a single extended attribute entry.
type fileFullEaInformation struct {
	NextEntryOffset uint32
	Flags           byte
	EaNameLength    byte
	EaValueLength   uint16
	// EaName follows this struct in memory.
}

var (
	modntdll          = windows.NewLazyDLL("ntdll.dll")
	procNtQueryEaFile = modntdll.NewProc("NtQueryEaFile")
)

// readExtendedAttributes reads all extended attributes from a file or directory.
// This is a best-effort operation; it returns nil if attributes cannot be read
// for any reason (e.g., permissions, no attributes present).
func readExtendedAttributes(path string) map[string]string {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}

	// FILE_FLAG_BACKUP_SEMANTICS is required to open directories.
	handle, err := windows.CreateFile(
		pathPtr,
		windows.FILE_READ_EA,
		windows.FILE_SHARE_READ|
			windows.FILE_SHARE_WRITE|
			windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return nil // Failed to open file/directory.
	}
	defer windows.CloseHandle(handle)

	var statusBlock ioStatusBlock
	buffer := make([]byte, initialEABufferSize)
	var ntStatus uintptr

	// Query for Extended Attributes. This may need to be attempted twice if the initial
	// buffer is too small.
	for {
		// Parameters for NtQueryEaFile:
		// - Return all EAs (SingleEntry=false)
		// - Start scan from the beginning (RestartScan=true)
		// - No filtering by name (EaList=nil)
		ntStatus, _, _ = procNtQueryEaFile.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&statusBlock)),
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(len(buffer)),
			0, // SingleEntry = FALSE
			0, // EaList = NULL
			0, // EaListLength = 0
			0, // EaIndex = NULL
			1, // RestartScan = TRUE
		)

		if ntStatus != statusBufferTooSmall {
			// Success or an unrecoverable error occurred, so we break the loop.
			break
		}

		// The buffer was too small. The required size is in statusBlock.Information.
		newSize := len(buffer) * 2
		if newSize > maxEABufferSize {
			return nil // Required size is too large.
		}
		buffer = make([]byte, newSize)
	}

	switch ntStatus {
	case statusSuccess:
		// Data was read successfully, now parse it.
		// The actual length of the data is in statusBlock.Information.
		return parseExtendedAttributes(buffer[:statusBlock.Information])
	case statusNoEAsOnFile:
		// The file has no EAs, which is not an error.
		return make(map[string]string)
	default:
		// Any other status indicates an error.
		return nil
	}
}

// parseExtendedAttributes decodes the raw byte buffer from NtQueryEaFile into a map.
// The buffer contains a sequence of fileFullEaInformation structs, each followed by
// the attribute's name and value.
func parseExtendedAttributes(data []byte) map[string]string {
	eas := make(map[string]string)
	if len(data) == 0 {
		return eas
	}

	var offset uint32
	for {
		// Cast the current position in the buffer to our EA info struct.
		eaInfo := (*fileFullEaInformation)(unsafe.Pointer(&data[offset]))

		// The name starts immediately after the fixed-size struct header.
		// A null terminator follows the name. The value starts after that.
		nameStart := offset + uint32(unsafe.Sizeof(fileFullEaInformation{}))
		nameEnd := nameStart + uint32(eaInfo.EaNameLength)
		valueStart := nameEnd + 1 // Skip the null terminator for the name
		valueEnd := valueStart + uint32(eaInfo.EaValueLength)

		// Boundary check to prevent a panic on malformed EA data.
		if valueEnd > uint32(len(data)) {
			break
		}

		name := strings.ToLower(string(data[nameStart:nameEnd]))
		value := convertEAValueToString(data[valueStart:valueEnd])
		eas[name] = value

		// If NextEntryOffset is 0, this is the last entry in the list.
		if eaInfo.NextEntryOffset == 0 {
			break
		}
		offset += eaInfo.NextEntryOffset
	}

	return eas
}

// convertEAValueToString interprets the EA value as a printable string.
// If the value contains non-printable characters or is not valid UTF-8,
// it returns a hex string representation (e.g., "0xDEADBEEF").
func convertEAValueToString(value []byte) string {
	if len(value) == 0 {
		return ""
	}

	// Check if the byte slice is valid UTF-8.
	if utf8.Valid(value) {
		// Trim trailing null bytes, which are common.
		str := string(bytes.TrimRight(value, "\x00"))

		// Check for non-printable characters (control codes), allowing common whitespace.
		isPrintable := strings.IndexFunc(str, func(r rune) bool {
			var (
				isControl = r < 32
				isTab     = r == '\t'
				isCR      = r == '\r'
				isLF      = r == '\n'
			)
			return isControl && !isTab && !isCR && !isLF
		}) == -1

		if isPrintable {
			return str
		}
	}

	// Fallback to a hexadecimal representation for non-string data.
	return fmt.Sprintf("0x%X", value)
}
