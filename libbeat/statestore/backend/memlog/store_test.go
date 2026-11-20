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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
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

// TestOpenStoreRecoversMissingMeta verifies that openStore creates meta.json when
// the store directory already exists but meta.json is absent — the scenario that
// arises when the process crashes after MkdirAll but before writeMetaFile on a
// prior run.
func TestOpenStoreRecoversMissingMeta(t *testing.T) {
	dir := t.TempDir()
	// Simulate a crash after MkdirAll: the directory exists but is empty.
	storePath := filepath.Join(dir, "store")
	require.NoError(t, os.Mkdir(storePath, 0o770))

	logger := logptest.NewTestingLogger(t, "")
	store, err := openStore(logger.Named("test"), storePath, 0o660, 4096, false, 0, func(_ uint64) bool {
		return false
	})
	require.NoError(t, err, "openStore must succeed even when meta.json was missing")
	store.Close()

	require.FileExists(t, filepath.Join(storePath, metaFileName), "meta.json must be created on recovery")

	meta, err := readMetaFile(storePath)
	require.NoError(t, err)
	assert.NoError(t, checkMeta(meta))
}

func TestRecoverFromCorruption(t *testing.T) {
	path := t.TempDir()

	if err := copyPath(path, "testdata/1/logfile_incomplete/"); err != nil {
		t.Fatalf("Failed to copy test file to the temporary directory: %v", err)
	}

	logger := logptest.NewTestingLogger(t, "")
	store, err := openStore(logger.Named("test"), path, 0660, 4096, false, 0, func(_ uint64) bool {
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

func TestCheckpointThreshold(t *testing.T) {
	const mib = uint64(1 << 20)

	tests := map[string]struct {
		maxLogSize     uint64
		activeFileSize uint64
		want           uint64
	}{
		"no checkpoint uses the minimum": {
			maxLogSize: DefaultMaxLogSize,
			want:       DefaultMinLogSize,
		},
		"zero max log size falls back to default max": {
			activeFileSize: 100 * mib,
			want:           100 * mib,
		},
		"checkpoint smaller than minimum uses the minimum": {
			maxLogSize:     DefaultMaxLogSize,
			activeFileSize: 3 * mib,
			want:           DefaultMinLogSize,
		},
		"checkpoint above the minimum grows the threshold": {
			maxLogSize:     DefaultMaxLogSize,
			activeFileSize: 25 * mib,
			want:           25 * mib,
		},
		"threshold is capped at the default max": {
			maxLogSize:     DefaultMaxLogSize,
			activeFileSize: 2048 * mib,
			want:           DefaultMaxLogSize,
		},
		"threshold is capped at a custom max": {
			maxLogSize:     64 * mib,
			activeFileSize: 100 * mib,
			want:           64 * mib,
		},
		"custom max above the minimum and below the checkpoint": {
			maxLogSize:     20 * mib,
			activeFileSize: 5 * mib,
			want:           DefaultMinLogSize,
		},
		"custom max below the minimum wins": {
			maxLogSize:     5 * mib,
			activeFileSize: 0,
			want:           5 * mib,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := &diskstore{
				maxLogSize:     tc.maxLogSize,
				activeDataFile: dataFileInfo{size: tc.activeFileSize},
			}
			assert.Equal(t, tc.want, s.checkpointThreshold(), "unexpected checkpoint threshold")

			s.logFileSize = tc.want - 1
			assert.False(t, s.mustCheckpoint(), "log below the threshold must not checkpoint")
			s.logFileSize = tc.want
			assert.True(t, s.mustCheckpoint(), "log at the threshold must checkpoint")
		})
	}
}

func TestMustCheckpointCustomPredicateTakesPrecedence(t *testing.T) {
	s := &diskstore{
		maxLogSize:     DefaultMaxLogSize,
		checkpointPred: func(uint64) bool { return false },
		logFileSize:    DefaultMaxLogSize * 2,
	}
	assert.False(t, s.mustCheckpoint(), "custom predicate must override the dynamic threshold")

	s.checkpointPred = func(uint64) bool { return true }
	s.logFileSize = 0
	assert.True(t, s.mustCheckpoint(), "custom predicate must override the dynamic threshold")

	s.checkpointPred = func(uint64) bool { return false }
	s.logInvalid = true
	assert.True(t, s.mustCheckpoint(), "invalid log must always checkpoint")
}

func TestDynamicCheckpointWithMaxLogSize(t *testing.T) {
	const maxLogSize = 2 * 1024 // far below DefaultMinLogSize, so it caps the threshold

	home := filepath.Join(t.TempDir(), "store")
	logger := logptest.NewTestingLogger(t, "")

	store, err := openStore(logger.Named("test"), home, 0o660, 4096, false, maxLogSize, nil)
	require.NoError(t, err, "openStore must succeed")
	require.Zero(t, store.disk.activeDataFile.size, "new store must not have a checkpoint")
	require.Equal(t, uint64(maxLogSize), store.disk.checkpointThreshold(), "threshold must be capped by maxLogSize")

	// Write until the log reaches the threshold and a checkpoint is triggered.
	value := mapstr.M{"field": strings.Repeat("x", 128)}
	var checkpointed bool
	for i := 0; i < 100 && !checkpointed; i++ {
		require.NoError(t, store.Set(fmt.Sprintf("key-%d", i), value), "Set must succeed")
		checkpointed = store.disk.activeDataFile.path != ""
	}
	require.True(t, checkpointed, "a checkpoint must be triggered once the log reaches maxLogSize")

	active := store.disk.activeDataFile
	assert.Zero(t, store.disk.logFileSize, "log must be reset after the checkpoint")
	info, err := os.Stat(active.path)
	require.NoError(t, err, "Stat on the checkpoint file must succeed")
	assert.Equal(t, uint64(info.Size()), active.size, "activeDataFile.size must match the checkpoint size on disk") //nolint:gosec // test
	assert.Positive(t, active.size, "checkpoint size must be tracked")
	require.NoError(t, store.Close(), "Close must succeed")

	// Reopening must load the checkpoint size from disk.
	store, err = openStore(logger.Named("test"), home, 0o660, 4096, false, maxLogSize, nil)
	require.NoError(t, err, "re-open must succeed")
	defer store.Close()
	assert.Equal(t, active.path, store.disk.activeDataFile.path, "unexpected active data file after re-open")
	assert.Equal(t, active.size, store.disk.activeDataFile.size, "activeDataFile.size must be loaded from disk on re-open")
	assert.Equal(t, uint64(maxLogSize), store.disk.checkpointThreshold(), "threshold must stay capped by maxLogSize")
}

func TestDynamicCheckpointThresholdGrowsWithCheckpoint(t *testing.T) {
	home := filepath.Join(t.TempDir(), "store")
	logger := logptest.NewTestingLogger(t, "")

	store, err := openStore(logger.Named("test"), home, 0o660, 4096, false, DefaultMaxLogSize, nil)
	require.NoError(t, err, "openStore must succeed")
	require.Equal(t, DefaultMinLogSize, store.disk.checkpointThreshold(), "initial threshold must be DefaultMinLogSize")

	// Simulate a large checkpoint without writing megabytes of data.
	store.disk.activeDataFile.size = 25 << 20
	assert.Equal(t, uint64(25<<20), store.disk.checkpointThreshold(), "threshold must follow the checkpoint size")
	store.disk.activeDataFile.size = 4 << 30
	assert.Equal(t, DefaultMaxLogSize, store.disk.checkpointThreshold(), "threshold must be capped at DefaultMaxLogSize")
	require.NoError(t, store.Close(), "Close must succeed")
}

func TestListDataFilesReportsSize(t *testing.T) {
	home := t.TempDir()
	content := []byte(`[{"_key":"a"}]`)
	require.NoError(t, os.WriteFile(filepath.Join(home, "5.json"), content, 0o600), "write data file")

	infos, err := listDataFiles(home)
	require.NoError(t, err, "listDataFiles must succeed")
	require.Len(t, infos, 1, "expecting one data file")
	assert.Equal(t, uint64(len(content)), infos[0].size, "size must be populated from the file on disk")
}
