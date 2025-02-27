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

package testcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewFileWith644Perm creates a copy of the file at the given path in a
// temporary directory with 0644 permissions (before umask) and returns the path
// to the new file.
// Git does not preserve the full permissions of a file. Therefore, depending
// on the system's umask, the configuration file might have overly broad
// permissions, which could cause Beats to fail to start.
func NewFileWith644Perm(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read file %s", path)

	cfgPath := filepath.Join(t.TempDir(), filepath.Base(path))
	err = os.WriteFile(cfgPath, content, 0644)
	require.NoError(t, err, "failed to write temporary file: %v", err)

	return cfgPath
}

// CopyDirectoryWithOwnerWriteOnly creates a deep copy of the source directory
// src to a temporary location (t.TempDir()).
// The copied files have their group and others write permissions removed
// (chmod go-w equivalent).
//
// The prefix argument allows specifying a custom name for the copied directory
// within the temporary location. If prefix is empty, the base name of the
// source directory is used.
//
// Example:
//
//	src := "/path/to/source/directory"
//	dest := CopyDirectoryWithOwnerWriteOnly(t, src, "copied_directory")
//	// dest will be "/tmp/some_temp_dir/copied_directory"
//
//	src := "/path/to/source/directory"
//	dest := CopyDirectoryWithOwnerWriteOnly(t, src, "")
//	// dest will be "/tmp/some_temp_dir/directory"
//
// Note:
//
//	Directories are created with their original permissions.
//	Regular files are created with the same permissions as the source, except
//	that group and others write permissions are removed.
//
// Git does not preserve the full permissions of a file. Therefore, depending
// on the system's umask, the configuration file might have overly broad
// permissions, which could cause Beats to fail to start.
func CopyDirectoryWithOwnerWriteOnly(t *testing.T, src, prefix string) string {
	t.Helper()
	tempDir := t.TempDir()

	if prefix == "" {
		prefix = filepath.Base(src)
	}
	destPath := filepath.Join(tempDir, prefix)

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		dest := filepath.Join(destPath, path[len(src):])

		if info.IsDir() {
			return os.MkdirAll(dest, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dest, data, info.Mode()&^0022)
	})

	require.NoError(t, err, "failed to copy %s to %s to set permission to 0644",
		src, destPath)
	return destPath
}
