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

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSafeFileRotate creates two files, dest and src, and calls
// SafeFileRotate to replace dest with src. However, before the test
// makes that call, it deliberately keeps a handle open on dest for
// a short period of time to ensure that the rotation takes place anyway
// after the handle has been released.
func TestSafeFileRotate(t *testing.T) {
	// Create destination file
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "dest.txt")
	err := os.WriteFile(dest, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Create source file
	src := filepath.Join(tmpDir, "src.txt")
	err = os.WriteFile(src, []byte("new content"), 0644)
	require.NoError(t, err)

	// Open handle on dest file for 1.5 seconds
	destFile, err := os.Open(dest)
	require.NoError(t, err)
	time.AfterFunc(1500*time.Millisecond, func() {
		destFile.Close() // Close the handle after 1.5 seconds
	})
	defer destFile.Close()

	// Try to replace dest with new
	err = SafeFileRotate(dest, src, WithRenameRetries(2*time.Second, 100*time.Millisecond))
	require.NoError(t, err)

	// Check that dest file has been replaced with new file
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "new content", string(data))
}
