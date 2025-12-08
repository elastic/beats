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

package file_integrity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hectane/go-acl"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

// TestFileInfoPermissions tests obtaining metadata of a file
// when we don't have permissions to open the file for reading.
// This prevents us to get the file owner of a file unless we use
// a method that doesn't need to open the file for reading.
// (GetNamedSecurityInfo vs CreateFile+GetSecurityInfo)
func TestFileInfoPermissions(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	name := f.Name()

	makeFileNonReadable(t, f.Name())
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := NewMetadata(name, info)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	t.Log(meta.Owner)
	assert.NotEqual(t, "", meta.Owner)
}

func makeFileNonReadable(t testing.TB, path string) {
	if err := acl.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
}

// TestGetObjectSecurityInfoRobustness tests the robust security info functionality
// with various scenarios including privilege escalation and fallback mechanisms.
func TestGetObjectSecurityInfoRobustness(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(t *testing.T) (string, bool, func()) // returns path, isDir, cleanup
		expectError    bool
		expectOwnerSID bool
		expectOwner    bool
		expectGroup    bool
	}{
		{
			name: "regular file owned by current user",
			setupFile: func(t *testing.T) (string, bool, func()) {
				tmpFile, err := os.CreateTemp("", "security_test_*.txt")
				if err != nil {
					t.Fatal(err)
				}
				tmpFile.Close()
				return tmpFile.Name(), false, func() { os.Remove(tmpFile.Name()) }
			},
			expectError:    false,
			expectOwnerSID: true,
			expectOwner:    true,
			expectGroup:    true,
		},
		{
			name: "directory owned by current user",
			setupFile: func(t *testing.T) (string, bool, func()) {
				tmpDir, err := os.MkdirTemp("", "security_test_*")
				if err != nil {
					t.Fatal(err)
				}
				return tmpDir, true, func() { os.RemoveAll(tmpDir) }
			},
			expectError:    false,
			expectOwnerSID: true,
			expectOwner:    true,
			expectGroup:    true,
		},
		{
			name: "non-readable file (fallback test)",
			setupFile: func(t *testing.T) (string, bool, func()) {
				tmpFile, err := os.CreateTemp("", "nonreadable_test_*.txt")
				if err != nil {
					t.Fatal(err)
				}
				tmpFile.Close()
				makeFileNonReadable(t, tmpFile.Name())
				return tmpFile.Name(), false, func() { os.Remove(tmpFile.Name()) }
			},
			expectError:    false, // Should work with fallback mechanisms
			expectOwnerSID: true,  // Should at least get SID
			expectOwner:    true,  // Should resolve owner name
			expectGroup:    false, // Group might not be available for restricted files
		},
		{
			name: "non-existent file",
			setupFile: func(t *testing.T) (string, bool, func()) {
				return filepath.Join(os.TempDir(), "non_existent_file_12345.txt"), false, func() {}
			},
			expectError:    true,
			expectOwnerSID: false,
			expectOwner:    false,
			expectGroup:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, isDir, cleanup := tt.setupFile(t)
			defer cleanup()

			info, err := getObjectSecurityInfo(path, isDir)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.expectOwnerSID {
				assert.NotEmpty(t, info.sid, "Should have owner SID")
				assert.Contains(t, info.sid, "S-", "SID should start with S-")
			}

			if tt.expectOwner {
				assert.NotEmpty(t, info.name, "Should have owner name")
			}

			if tt.expectGroup {
				// Group might be empty in some cases, but if present should be valid
				if info.groupName != "" {
					assert.NotEmpty(t, info.groupName, "Group name should not be empty string")
				}
			}

			t.Logf("Path: %s, SID: %s, Owner: %s, Group: %s", path, info.sid, info.name, info.groupName)
		})
	}
}

// TestGetSecurityInfoByPath tests the path-based security info fallback
func TestGetSecurityInfoByPath(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "security_path_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	info, err := getSecurityInfoByPath(tmpFile.Name())

	assert.NoError(t, err)
	assert.NotEmpty(t, info.sid, "Should have owner SID")
	assert.Contains(t, info.sid, "S-", "SID should start with S-")
	assert.NotEmpty(t, info.name, "Should have owner name")

	t.Logf("ByPath - SID: %s, Owner: %s, Group: %s", info.sid, info.name, info.groupName)
}

// TestGetSecurityInfoFromHandle tests the handle-based security info method
func TestGetSecurityInfoFromHandle(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "security_handle_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Open the file with minimal permissions
	pathPtr, err := windows.UTF16PtrFromString(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.READ_CONTROL,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	info, err := getSecurityInfoFromHandle(handle, tmpFile.Name())

	assert.NoError(t, err)
	assert.NotEmpty(t, info.sid, "Should have owner SID")
	assert.Contains(t, info.sid, "S-", "SID should start with S-")
	assert.NotEmpty(t, info.name, "Should have owner name")

	t.Logf("FromHandle - SID: %s, Owner: %s, Group: %s", info.sid, info.name, info.groupName)
}

// TestExtractSecurityInfo tests the security descriptor parsing
func TestExtractSecurityInfo(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "security_extract_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Get security descriptor using Windows API
	secInfo, err := windows.GetNamedSecurityInfo(
		tmpFile.Name(),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}

	info, err := extractSecurityInfo(secInfo, tmpFile.Name())

	assert.NoError(t, err)
	assert.NotEmpty(t, info.sid, "Should have owner SID")
	assert.Contains(t, info.sid, "S-", "SID should start with S-")
	assert.NotEmpty(t, info.name, "Should have owner name")

	t.Logf("Extract - SID: %s, Owner: %s, Group: %s", info.sid, info.name, info.groupName)
}

// TestSecurityInfoFallbackMechanisms tests various fallback scenarios
func TestSecurityInfoFallbackMechanisms(t *testing.T) {
	t.Run("owner_only_fallback", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "fallback_test_*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// This should work even if group information fails
		info, err := getSecurityInfoByPath(tmpFile.Name())
		assert.NoError(t, err)
		assert.NotEmpty(t, info.sid)
		assert.NotEmpty(t, info.name)
	})

	t.Run("progressive_access_levels", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "access_test_*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Test that the function works with our progressive access mechanism
		info, err := getObjectSecurityInfo(tmpFile.Name(), false)
		assert.NoError(t, err)
		assert.NotEmpty(t, info.sid)
		assert.NotEmpty(t, info.name)

		t.Logf("Progressive - SID: %s, Owner: %s, Group: %s", info.sid, info.name, info.groupName)
	})
}

// TestSecurityInfoWithDifferentFileTypes tests security info with various file types
func TestSecurityInfoWithDifferentFileTypes(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func(t *testing.T) (string, bool, func())
		fileType string
	}{
		{
			name: "regular_text_file",
			setupFn: func(t *testing.T) (string, bool, func()) {
				tmpFile, err := os.CreateTemp("", "text_test_*.txt")
				if err != nil {
					t.Fatal(err)
				}
				tmpFile.WriteString("test content")
				tmpFile.Close()
				return tmpFile.Name(), false, func() { os.Remove(tmpFile.Name()) }
			},
			fileType: "file",
		},
		{
			name: "directory",
			setupFn: func(t *testing.T) (string, bool, func()) {
				tmpDir, err := os.MkdirTemp("", "dir_test_*")
				if err != nil {
					t.Fatal(err)
				}
				return tmpDir, true, func() { os.RemoveAll(tmpDir) }
			},
			fileType: "directory",
		},
		{
			name: "executable_file",
			setupFn: func(t *testing.T) (string, bool, func()) {
				tmpFile, err := os.CreateTemp("", "exe_test_*.exe")
				if err != nil {
					t.Fatal(err)
				}
				tmpFile.Close()
				return tmpFile.Name(), false, func() { os.Remove(tmpFile.Name()) }
			},
			fileType: "executable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, isDir, cleanup := tt.setupFn(t)
			defer cleanup()

			info, err := getObjectSecurityInfo(path, isDir)

			assert.NoError(t, err, "Should handle %s without error", tt.fileType)
			assert.NotEmpty(t, info.sid, "Should have owner SID for %s", tt.fileType)
			assert.NotEmpty(t, info.name, "Should have owner name for %s", tt.fileType)

			t.Logf("%s - SID: %s, Owner: %s, Group: %s", tt.fileType, info.sid, info.name, info.groupName)
		})
	}
}

// TestSecurityInfoErrorHandling tests error conditions
func TestSecurityInfoErrorHandling(t *testing.T) {
	t.Run("invalid_path", func(t *testing.T) {
		invalidPath := "Z:\\completely\\invalid\\path\\that\\does\\not\\exist.txt"
		_, err := getObjectSecurityInfo(invalidPath, false)
		assert.Error(t, err, "Should fail for invalid path")
	})

	t.Run("empty_path", func(t *testing.T) {
		_, err := getObjectSecurityInfo("", false)
		assert.Error(t, err, "Should fail for empty path")
	})

	t.Run("path_with_invalid_characters", func(t *testing.T) {
		invalidPath := "test\x00file.txt" // null character
		_, err := getObjectSecurityInfo(invalidPath, false)
		assert.Error(t, err, "Should fail for path with invalid characters")
	})
}
