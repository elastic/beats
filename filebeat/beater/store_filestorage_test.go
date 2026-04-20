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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestOpenStateStore_OtelFileStorageSetGet(t *testing.T) {
	dir := t.TempDir()
	beatPaths := paths.New()
	beatPaths.Data = dir

	fb, err := openStateStore(t.Context(), beat.Info{Beat: "testbeat"}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Permissions:   0o600,
		CleanInterval: 5 * time.Second,
		Backend:       "otel_file_storage",
	}, beatPaths)
	require.NoError(t, err)
	defer fb.Close()

	st, err := fb.StoreFor("filestream")
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

	_, err := openStateStore(t.Context(), beat.Info{Beat: "test"}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Backend:       "not_a_real_backend",
		CleanInterval: 5 * time.Second,
	}, beatPaths)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown registry backend")
}

func TestOpenStateStore_OtelFileStorageInvalidBeatName(t *testing.T) {
	dir := t.TempDir()
	beatPaths := paths.New()
	beatPaths.Data = dir

	_, err := openStateStore(t.Context(), beat.Info{Beat: "!!!invalid component id!!!"}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Backend:       "otel_file_storage",
		CleanInterval: 5 * time.Second,
	}, beatPaths)
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

	s1, err := openStateStore(t.Context(), beat.Info{Beat: "a"}, logp.NewNopLogger(), cfg, beatPaths)
	require.NoError(t, err)
	defer s1.Close()

	s2, err := openStateStore(t.Context(), beat.Info{Beat: "b"}, logp.NewNopLogger(), cfg, beatPaths)
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
