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
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procNtSetEaFile = modntdll.NewProc("NtSetEaFile")

// TestMain runs before all other tests in this package.
// It checks for the host requirement of an NTFS filesystem.
func TestMain(m *testing.M) {
	// Create a temporary directory to check its underlying filesystem.
	tempDir, err := os.MkdirTemp("", "test-ea-")
	if err != nil {
		fmt.Println("Failed to create temp dir for filesystem check:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	isNTFS, err := isHostFSNTFS(tempDir)
	if err != nil {
		fmt.Printf("Could not determine filesystem type: %v. Skipping tests.\n", err)
		os.Exit(0) // Skip tests if we can't be sure.
	}

	if !isNTFS {
		fmt.Println("Host filesystem is not NTFS. Skipping extended attribute tests.")
		os.Exit(0) // Gracefully skip all tests.
	}

	// Run the actual tests.
	os.Exit(m.Run())
}

// isHostFSNTFS checks if the filesystem for the given path is NTFS.
// Extended Attributes are only supported on NTFS.
func isHostFSNTFS(path string) (bool, error) {
	root := filepath.VolumeName(path)
	if root == "" {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return false, err
		}
		root = filepath.VolumeName(path)
	}
	root = strings.TrimRight(root, `\`) // Ensure no trailing slash for UNC paths

	rootPtr, err := windows.UTF16PtrFromString(root + `\`)
	if err != nil {
		return false, err
	}

	fsNameBuffer := make([]uint16, windows.MAX_PATH)
	err = windows.GetVolumeInformation(rootPtr, nil, 0, nil, nil, nil, &fsNameBuffer[0], uint32(len(fsNameBuffer)))
	if err != nil {
		return false, err
	}

	fsName := windows.UTF16ToString(fsNameBuffer)
	return fsName == "NTFS", nil
}

// TestReadExtendedAttributes is a table-driven test for the readExtendedAttributes function.
func TestReadExtendedAttributes(t *testing.T) {
	// Helper function to create a temporary file.
	createTempFile := func(t *testing.T, dir string) *os.File {
		t.Helper()
		f, err := os.CreateTemp(dir, "testfile-")
		if err != nil {
			t.Fatal(err)
		}
		return f
	}

	testCases := []struct {
		name      string
		setupFunc func(t *testing.T, path string) // Sets up the file/dir with EAs.
		isDir     bool
		expected  map[string]string
		expectNil bool
	}{
		{
			name: "Success with multiple EAs",
			setupFunc: func(t *testing.T, path string) {
				eas := map[string][]byte{
					"user.app.id":    []byte("app.12345"),
					"user.mime.type": []byte("text/plain"),
				}
				if err := setExtendedAttributes(path, eas); err != nil {
					t.Fatalf("Failed to set EAs for test: %v", err)
				}
			},
			expected: map[string]string{
				"user.app.id":    "app.12345",
				"user.mime.type": "text/plain",
			},
		},
		{
			name:      "File with no EAs",
			setupFunc: func(t *testing.T, path string) {}, // No EAs to set.
			expected:  nil,
		},
		{
			name: "Non-printable value",
			setupFunc: func(t *testing.T, path string) {
				eas := map[string][]byte{"user.binary": {0xDE, 0xAD, 0xBE, 0xEF}}
				if err := setExtendedAttributes(path, eas); err != nil {
					t.Fatalf("Failed to set EAs for test: %v", err)
				}
			},
			expected: map[string]string{
				"user.binary": "0xDEADBEEF",
			},
		},
		{
			name: "Large EA value to trigger buffer resize",
			setupFunc: func(t *testing.T, path string) {
				// Value is larger than initialEABufferSize (4096).
				largeValue := make([]byte, initialEABufferSize+100)
				for i := range largeValue {
					largeValue[i] = 'A'
				}
				eas := map[string][]byte{"user.large": largeValue}
				if err := setExtendedAttributes(path, eas); err != nil {
					t.Fatalf("Failed to set EAs for test: %v", err)
				}
			},
			expected: map[string]string{
				"user.large": string(make([]byte, initialEABufferSize+100)), // Filled with 'A's, but recreate for comparison.
			},
		},
		{
			name:      "File does not exist",
			setupFunc: func(t *testing.T, path string) { os.Remove(path) }, // Ensure file is gone.
			expectNil: true,
		},
		{
			name:  "Directory with EAs",
			isDir: true,
			setupFunc: func(t *testing.T, path string) {
				eas := map[string][]byte{"user.dir.meta": []byte("metadata for directory")}
				if err := setExtendedAttributes(path, eas); err != nil {
					t.Fatalf("Failed to set EAs for test: %v", err)
				}
			},
			expected: map[string]string{
				"user.dir.meta": "metadata for directory",
			},
		},
	}
	// The large EA test string needs to be filled with 'A's.
	testCases[3].expected["user.large"] = strings.Repeat("A", initialEABufferSize+100)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			var path string
			if tc.isDir {
				path = filepath.Join(tempDir, "testdir")
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			} else {
				f := createTempFile(t, tempDir)
				path = f.Name()
				f.Close()
			}

			if tc.setupFunc != nil {
				tc.setupFunc(t, path)
			}

			result := readExtendedAttributes(path)

			if tc.expectNil {
				if result != nil {
					t.Errorf("Expected nil result, but got %v", result)
				}
				return
			}

			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Mismatch in extended attributes.\nGot:      %v\nExpected: %v", result, tc.expected)
			}
		})
	}
}

// setExtendedAttributes is a test helper to write EAs to a file or directory.
// It uses the native NtSetEaFile function.
func setExtendedAttributes(path string, eas map[string][]byte) error {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}

	// Open handle with backup semantics to support directories.
	// We need WRITE_EA access to set attributes.
	handle, err := windows.CreateFile(
		pathPtr,
		windows.FILE_WRITE_EA,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return fmt.Errorf("CreateFile failed: %w", err)
	}
	defer windows.CloseHandle(handle)

	eaBuffer, err := buildEaBuffer(eas)
	if err != nil {
		return fmt.Errorf("failed to build EA buffer: %w", err)
	}
	if len(eaBuffer) == 0 {
		return nil // Nothing to set.
	}

	var statusBlock ioStatusBlock
	ntStatus, _, _ := procNtSetEaFile.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&statusBlock)),
		uintptr(unsafe.Pointer(&eaBuffer[0])),
		uintptr(len(eaBuffer)),
	)

	if ntStatus != statusSuccess {
		return fmt.Errorf("NtSetEaFile failed with status 0x%X", ntStatus)
	}
	return nil
}

// buildEaBuffer serializes a map of EAs into the binary format required by NtSetEaFile.
func buildEaBuffer(eas map[string][]byte) ([]byte, error) {
	if len(eas) == 0 {
		return nil, nil
	}

	// Sort keys for deterministic order, which is crucial for calculating offsets.
	keys := make([]string, 0, len(eas))
	for k := range eas {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer

	for i, name := range keys {
		value := eas[name]
		nameBytes := []byte(name)

		// Pre-validate inputs.
		if len(nameBytes) > 255 {
			return nil, fmt.Errorf("EA name too long: %s", name)
		}
		if len(value) > 65535 {
			return nil, fmt.Errorf("EA value too long for name: %s", name)
		}

		// The total size of this entry (header + name + null + value).
		entrySize := unsafe.Sizeof(fileFullEaInformation{}) + uintptr(len(nameBytes)) + 1 + uintptr(len(value))

		// The offset to the *next* entry must be aligned to a 4-byte boundary.
		alignedOffset := (entrySize + 3) &^ 3

		var nextOffset uint32
		// If this is not the last entry, set the offset. Otherwise, it's 0.
		if i < len(keys)-1 {
			nextOffset = uint32(alignedOffset)
		}

		info := fileFullEaInformation{
			NextEntryOffset: nextOffset, // Correctly set to 0 for the last entry.
			Flags:           0,
			EaNameLength:    byte(len(nameBytes)),
			EaValueLength:   uint16(len(value)),
		}

		// Write the header, name, null terminator, and value.
		buf.Write((*[unsafe.Sizeof(info)]byte)(unsafe.Pointer(&info))[:])
		buf.Write(nameBytes)
		buf.WriteByte(0)
		buf.Write(value)

		// Add padding to the buffer to respect the alignment for the next entry.
		// This is only necessary if it's not the last entry.
		if i < len(keys)-1 {
			paddingSize := alignedOffset - entrySize
			if paddingSize > 0 {
				buf.Write(make([]byte, paddingSize))
			}
		}
	}

	return buf.Bytes(), nil
}
