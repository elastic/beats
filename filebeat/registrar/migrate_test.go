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

//go:build linux || darwin

package registrar

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeWriteFile(t *testing.T) {
	data := []byte(`{"version": "1"}`)
	perm := os.FileMode(0o600)

	t.Run("success", func(t *testing.T) {
		dir := tempDir(t)
		path := filepath.Join(dir, "meta.json")

		require.NoError(t, safeWriteFile(path, data, perm))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, data, got)

		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, perm, fi.Mode().Perm(), "file permissions")

		assertNoTempFiles(t, dir)
	})

	t.Run("parent_dir_missing", func(t *testing.T) {
		// os.CreateTemp fails because the directory does not exist.
		path := filepath.Join(tempDir(t), "nonexistent", "meta.json")

		err := safeWriteFile(path, data, perm)
		require.Error(t, err)
	})

	t.Run("rotate_fails_cleans_up_temp_file", func(t *testing.T) {
		dir := tempDir(t)
		// Place a directory at the target path so os.Rename returns EISDIR,
		// forcing SafeFileRotate to fail after the temp file has been written.
		path := filepath.Join(dir, "meta.json")
		mkDir(t, path)

		err := safeWriteFile(path, data, perm)
		require.Error(t, err)

		// The temp file must not be left behind despite the rotate failure.
		assertNoTempFiles(t, dir)
	})
}

// assertNoTempFiles fails the test if any temp files from safeWriteFile remain
// in dir. The naming pattern is <base>.tmp-<random>.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.tmp-*"))
	require.NoError(t, err)
	assert.Empty(t, matches, "unexpected temp files in %s", dir)
}
