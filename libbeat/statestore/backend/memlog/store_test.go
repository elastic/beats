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

package memlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestWriteMetaFileAtomic(t *testing.T) {
	t.Run("writes_valid_meta_and_cleans_up_temp", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, writeMetaFile(dir, 0o600))

		meta, err := readMetaFile(dir)
		require.NoError(t, err)
		require.NoError(t, checkMeta(meta))

		matches, err := filepath.Glob(filepath.Join(dir, "*.tmp-*"))
		require.NoError(t, err)
		assert.Empty(t, matches, "no temp files should remain after successful write")
	})

	t.Run("no_partial_file_on_rotate_failure", func(t *testing.T) {
		dir := t.TempDir()
		// Place a directory at the meta.json path so SafeFileRotate (rename) fails.
		// This simulates a crash-safe scenario: the destination is never touched
		// until the temp file is fully written and synced.
		metaPath := filepath.Join(dir, metaFileName)
		require.NoError(t, os.Mkdir(metaPath, 0o700))

		err := writeMetaFile(dir, 0o600)
		require.Error(t, err)

		// The directory at the destination must be intact (not replaced or emptied).
		fi, statErr := os.Stat(metaPath)
		require.NoError(t, statErr)
		assert.True(t, fi.IsDir(), "destination should still be a directory after failed rotate")

		// No temp files should remain.
		matches, globErr := filepath.Glob(filepath.Join(dir, "*.tmp-*"))
		require.NoError(t, globErr)
		assert.Empty(t, matches, "no temp files should remain after failed rotate")
	})
}

func TestRecoverFromCorruption(t *testing.T) {
	path := t.TempDir()

	if err := copyPath(path, "testdata/1/logfile_incomplete/"); err != nil {
		t.Fatalf("Failed to copy test file to the temporary directory: %v", err)
	}

	logger := logptest.NewTestingLogger(t, "")
	store, err := openStore(logger.Named("test"), path, 0660, 4096, false, func(_ uint64) bool {
		return false
	})
	require.NoError(t, err, "openStore must succeed")
	require.True(t, store.disk.logInvalid, "expecting the log file to be invalid")

	err = store.logOperation(&opSet{K: "key", V: mapstr.M{
		"field": 42,
	}})
	require.NoError(t, err, "logOperation must succeed")
	require.False(t, store.disk.logInvalid, "log file must be valid")
	require.FileExistsf(t, filepath.Join(path, "7.json"), "expecting the checkpoint file to have been created")

	file, err := os.Stat(filepath.Join(path, "log.json"))
	require.NoError(t, err, "Stat on the log file must succeed")
	require.Equal(t, int64(0), file.Size(), "expecting the log file to be truncated")
}
