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

package beater

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/paths"
)

func TestOpenStateStore_OtelFileStorageSetGet(t *testing.T) {
	dir := t.TempDir()
	beatPaths := paths.New()
	beatPaths.Data = dir

	fb, err := openStateStore(t.Context(), beat.Info{Beat: "testbeat", Paths: beatPaths}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Permissions:   0o600,
		CleanInterval: 5 * time.Second,
		Backend:       "otel_file_storage",
	})
	require.NoError(t, err)
	defer fb.Close()

	st, err := fb.StoreFor("filestream", "")
	require.NoError(t, err)
	defer st.Close()

	require.NoError(t, st.Set("k1", map[string]any{"n": "hello"}))
	var got map[string]any
	require.NoError(t, st.Get("k1", &got))
	assert.Equal(t, "hello", got["n"])
}

func TestOpenStateStore_UnknownBackend(t *testing.T) {
	dir := t.TempDir()
	beatPaths := paths.New()
	beatPaths.Data = dir

	_, err := openStateStore(t.Context(), beat.Info{Beat: "test", Paths: beatPaths}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Backend:       "not_a_real_backend",
		CleanInterval: 5 * time.Second,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown registry backend")
}

func TestOpenStateStore_OtelFileStorageInvalidBeatName(t *testing.T) {
	dir := t.TempDir()
	beatPaths := paths.New()
	beatPaths.Data = dir

	_, err := openStateStore(t.Context(), beat.Info{Beat: "!!!invalid component id!!!", Paths: beatPaths}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Backend:       "otel_file_storage",
		CleanInterval: 5 * time.Second,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid beat name for otel_file_storage registry")
}

func TestOpenStateStore_OtelFileStorageSharesRegistryByPath(t *testing.T) {
	dir := t.TempDir()
	beatPaths := paths.New()
	beatPaths.Data = dir

	cfg := config.Registry{
		Path:          "",
		Permissions:   0o600,
		CleanInterval: 5 * time.Second,
		Backend:       "otel_file_storage",
	}

	s1, err := openStateStore(t.Context(), beat.Info{Beat: "a", Paths: beatPaths}, logp.NewNopLogger(), cfg)
	require.NoError(t, err)
	defer s1.Close()

	s2, err := openStateStore(t.Context(), beat.Info{Beat: "b", Paths: beatPaths}, logp.NewNopLogger(), cfg)
	require.NoError(t, err)
	defer s2.Close()

	assert.Same(t, s1.shared, s2.shared)
}

func TestFilestorageConfigFromRegistry_Defaults(t *testing.T) {
	dir := t.TempDir()
	fsCfg, err := filestorageConfigFromRegistry(config.Registry{
		Permissions: 0o600,
	}, dir)
	require.NoError(t, err)

	assert.Equal(t, dir, fsCfg.Directory)
	assert.True(t, fsCfg.CreateDirectory)
	assert.Equal(t, "0700", fsCfg.DirectoryPermissions)
	assert.NotNil(t, fsCfg.Compaction)
}

func TestFilestorageConfigFromRegistry_ExplicitMap(t *testing.T) {
	dir := t.TempDir()
	fsCfg, err := filestorageConfigFromRegistry(config.Registry{
		Permissions: 0o600,
		FileStorage: map[string]any{
			"timeout":               "5s",
			"create_directory":      true,
			"directory_permissions": "0750",
			"fsync":                 true,
			"recreate":              true,
		},
	}, dir)
	require.NoError(t, err)

	assert.Equal(t, dir, fsCfg.Directory)
	assert.Equal(t, 5*time.Second, fsCfg.Timeout)
	assert.True(t, fsCfg.CreateDirectory)
	assert.Equal(t, "0750", fsCfg.DirectoryPermissions)
	assert.True(t, fsCfg.FSync)
	assert.True(t, fsCfg.Recreate)
	assert.NotNil(t, fsCfg.Compaction)
}

func TestFilestorageConfigFromRegistry_EmptyMap(t *testing.T) {
	dir := t.TempDir()
	fsCfg, err := filestorageConfigFromRegistry(config.Registry{
		Permissions: 0o600,
		FileStorage: map[string]any{},
	}, dir)
	require.NoError(t, err)

	assert.Equal(t, dir, fsCfg.Directory)
	assert.False(t, fsCfg.CreateDirectory)
	assert.NotNil(t, fsCfg.Compaction)
}

func TestFilestorageConfigFromRegistry_PartialMap(t *testing.T) {
	dir := t.TempDir()
	fsCfg, err := filestorageConfigFromRegistry(config.Registry{
		Permissions: 0o600,
		FileStorage: map[string]any{
			"create_directory": true,
		},
	}, dir)
	require.NoError(t, err)

	assert.Equal(t, dir, fsCfg.Directory)
	assert.True(t, fsCfg.CreateDirectory)
	assert.Equal(t, "0700", fsCfg.DirectoryPermissions)
	assert.False(t, fsCfg.FSync)
	assert.NotNil(t, fsCfg.Compaction)
}

// countCheckpointFiles returns the number of "<txid>.json" checkpoint files in dir.
func countCheckpointFiles(t *testing.T, dir string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "[0-9]*.json"))
	require.NoError(t, err, "glob checkpoint files")
	return len(matches)
}

func TestOpenStateStore_MemlogNegativeMaxLogSize(t *testing.T) {
	beatPaths := paths.New()
	beatPaths.Data = t.TempDir()

	_, err := openStateStore(t.Context(), beat.Info{Beat: "testbeat", Paths: beatPaths}, logp.NewNopLogger(), config.Registry{
		Path:          "registry",
		Permissions:   0o600,
		CleanInterval: 5 * time.Minute,
		Backend:       "memlog",
		MaxLogSize:    cfgtype.ByteSize(-1),
	})
	require.Error(t, err, "negative MaxLogSize must be rejected")
	assert.Contains(t, err.Error(), "invalid registry.max_log_size", "unexpected error message")
}

func TestOpenStateStore_MemlogMaxLogSize(t *testing.T) {
	tcs := map[string]struct {
		maxLogSize         cfgtype.ByteSize
		wantCheckpointFile bool
	}{
		"small max log size forces a checkpoint": {
			maxLogSize:         cfgtype.ByteSize(2 * 1024),
			wantCheckpointFile: true,
		},
		"unset max log size uses the memlog default": {
			maxLogSize:         0,
			wantCheckpointFile: false,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			beatPaths := paths.New()
			beatPaths.Data = dir

			fb, err := openStateStore(t.Context(), beat.Info{Beat: "testbeat", Paths: beatPaths}, logp.NewNopLogger(), config.Registry{
				Path:          "registry",
				Permissions:   0o600,
				CleanInterval: 5 * time.Minute,
				Backend:       "memlog",
				MaxLogSize:    tc.maxLogSize,
			})
			require.NoError(t, err, "openStateStore must succeed")
			defer fb.Close()

			st, err := fb.StoreFor("filestream", "")
			require.NoError(t, err, "StoreFor must succeed")
			defer st.Close()

			// ~50KiB of log data: well above 2KiB, well below the 10MiB minimum.
			for i := range 100 {
				require.NoError(t, st.Set(fmt.Sprintf("k%d", i), map[string]any{"n": strings.Repeat("x", 256)}), "Set must succeed")
			}

			got := countCheckpointFiles(t, filepath.Join(dir, "registry", "testbeat"))
			if tc.wantCheckpointFile {
				assert.Positive(t, got, "expecting MaxLogSize to be passed to memlog and trigger a checkpoint")
			} else {
				assert.Zero(t, got, "expecting no checkpoint with the default memlog max log size")
			}
		})
	}
}
