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
	"fmt"
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

const (
	filebeatStateKey          = "filebeat/registry_backups/state"
	filebeatBackupMetadataKey = "filebeat/registry_backups/backup"
	registryBackupSchemaV1    = 1
	defaultRegistryBackend    = "memlog"
)

type filebeatStateRecord struct {
	SchemaVersion   int             `json:"schema_version"`
	RegistryVersion uint64          `json:"registry_version"`
	RegistryBackend string          `json:"registry_backend"`
	Features        map[string]bool `json:"features"`
	FilebeatVersion string          `json:"filebeat_version"`
}

type backupMetadataRecord struct {
	SchemaVersion int                 `json:"schema_version"`
	BackupID      string              `json:"backup_id"`
	CreatedAt     time.Time           `json:"created_at"`
	Source        filebeatStateRecord `json:"source"`
	Registry      backupRegistry      `json:"registry"`
}

type backupRegistry struct {
	RelativePath string        `json:"relative_path"`
	Files        []backupFiles `json:"files"`
}

type backupFiles struct {
	Path   string
	size   uint64
	sha256 string
}

var featureFlags = []string{"LogInputRunFilestream"}

func handleBackup(ctx context.Context, logger *logp.Logger, store backend.BackupStore, reg config.Registry, beatsPaths *paths.Path) error {
	if store == nil {
		logger.Debug("backup storage is not configured, skipping backup metadata persistence")
		return nil
	}

	resolvedPath := beatsPaths.Resolve(paths.Data, reg.Path)
	currentState := newFilebeatStateRecord(reg)

	previousStateRaw, err := store.Get(ctx, filebeatStateKey)
	if err != nil {
		return fmt.Errorf("cannot read previous state: %w", err)
	}

	previousState := filebeatStateRecord{}
	if json.Unmarshal(previousStateRaw, &previousState); err != nil {
		return fmt.Errorf("cannot decode Filebeat backup state: %w", err)
	}

	if backupNeeded(previousState, currentState) {
		logger.Info("==================== BACKUP NEEDED")
		if err := writeBackup(store, currentState, reg); err != nil {
			return fmt.Errorf("cannot write backup: %w", err)
		}
	}

	logger.Infof("================================================== Store path: %q", resolvedPath)

	statePayload, err := json.Marshal(currentState)
	if err != nil {
		return fmt.Errorf("failed to marshal backup state record: %w", err)
	}
	if err := store.Set(context.Background(), filebeatStateKey, statePayload); err != nil {
		return fmt.Errorf("failed to store backup state record: %w", err)
	}

	return nil
}

func backupNeeded(prev, curr filebeatStateRecord) bool {
	if curr.RegistryVersion != prev.RegistryVersion {
		return true
	}

	for _, ff := range featureFlags {
		if curr.Features[ff] != prev.Features[ff] {
			return true
		}
	}

	return false
}

func writeBackup(store backend.BackupStore, currentState filebeatStateRecord, reg config.Registry) error {
	backupRecord := newBackupMetadataRecord(currentState, reg)
	backupPayload, err := json.Marshal(backupRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal backup metadata record: %w", err)
	}
	if err := store.Set(context.Background(), filebeatBackupMetadataKey, backupPayload); err != nil {
		return fmt.Errorf("failed to store backup metadata record: %w", err)
	}

	return nil
}

func newFilebeatStateRecord(reg config.Registry) filebeatStateRecord {
	filebeatVersion := version.GetDefaultVersion()

	return filebeatStateRecord{
		SchemaVersion:   registryBackupSchemaV1,
		RegistryVersion: backend.Version,
		RegistryBackend: reg.Backend,
		Features: map[string]bool{
			"LogInputRunFilestream": features.LogInputRunFilestream(),
		},
		FilebeatVersion: filebeatVersion,
	}
}

func newBackupMetadataRecord(source filebeatStateRecord, reg config.Registry) backupMetadataRecord {
	now := time.Now().UTC()

	return backupMetadataRecord{
		SchemaVersion: registryBackupSchemaV1,
		BackupID:      now.Format(time.RFC3339Nano),
		CreatedAt:     now,
		Source:        source,
		Registry: backupRegistry{
			RelativePath: reg.Path,
		},
	}
}
