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
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/version"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

type testStorage struct {
	values map[string][]byte
}

func (s *testStorage) Get(_ context.Context, key string) ([]byte, error) {
	return s.values[key], nil
}

func (s *testStorage) Set(_ context.Context, key string, value []byte) error {
	if s.values == nil {
		s.values = map[string][]byte{}
	}
	s.values[key] = append([]byte(nil), value...)
	return nil
}

func (s *testStorage) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestHandleBackupStoresStateAndMetadataRecords(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]any{
		"features": map[string]any{
			"log_input_run_as_filestream": map[string]any{
				"enabled": true,
			},
		},
	})
	require.NoError(t, err, "failed to build feature-flag config")
	require.NoError(t, features.UpdateFromConfig(cfg), "failed to set LogInputRunFilestream feature flag")
	t.Cleanup(func() {
		resetCfg, resetErr := conf.NewConfigFrom(map[string]any{
			"features": map[string]any{
				"log_input_run_as_filestream": map[string]any{
					"enabled": false,
				},
			},
		})
		require.NoError(t, resetErr, "failed to build feature-flag reset config")
		require.NoError(t, features.UpdateFromConfig(resetCfg), "failed to reset LogInputRunFilestream feature flag")
	})

	store := &testStorage{}
	beatPaths := paths.New()
	beatPaths.Data = t.TempDir()

	err = handleBackup(logp.NewNopLogger(), store, config.Registry{Path: "registry"}, beatPaths)
	require.NoError(t, err, "handleBackup should persist the backup metadata records")

	statePayload, ok := store.values[filebeatStateKey]
	require.True(t, ok, "state record must be stored under %q", filebeatStateKey)

	var stateRecord filebeatStateRecord
	require.NoError(t, json.Unmarshal(statePayload, &stateRecord), "state record must be valid JSON")
	assert.Equal(t, registryBackupSchemaV1, stateRecord.SchemaVersion, "state record must use the current schema version")
	assert.Equal(t, uint64(2), stateRecord.RegistryVersion, "state record must store the derived compatibility version")
	assert.Equal(t, defaultRegistryBackend, stateRecord.RegistryBackend, "state record must normalize the default registry backend")
	assert.Equal(t, version.GetDefaultVersion(), stateRecord.FilebeatVersion, "state record must store the current Filebeat version")
	assert.Equal(t, map[string]bool{"LogInputRunFilestream": true}, stateRecord.Features, "state record must store the relevant feature flags")

	backupPayload, ok := store.values[filebeatBackupMetadataKey]
	require.True(t, ok, "backup record must be stored under %q", filebeatBackupMetadataKey)

	var backupRecord backupMetadataRecord
	require.NoError(t, json.Unmarshal(backupPayload, &backupRecord), "backup record must be valid JSON")
	assert.Equal(t, registryBackupSchemaV1, backupRecord.SchemaVersion, "backup record must use the current schema version")
	assert.NotEmpty(t, backupRecord.BackupID, "backup record must contain a backup identifier")
	assert.False(t, backupRecord.CreatedAt.IsZero(), "backup record must contain a creation timestamp")
	assert.Equal(t, stateRecord, backupRecord.Source, "backup record must embed the source compatibility state")
	assert.Equal(t, "registry", backupRecord.Registry.RelativePath, "backup record must store the registry-relative path")
}

func TestHandleBackupSkipsWhenStorageIsNil(t *testing.T) {
	beatPaths := paths.New()
	beatPaths.Data = t.TempDir()

	err := handleBackup(logp.NewNopLogger(), nil, config.Registry{Path: "registry"}, beatPaths)
	require.NoError(t, err, "handleBackup should no-op when no backup storage is configured")
}
