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
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/features"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/bbolt"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

const (
	filebeatStateKey          = "filebeat/registry_backups/state"
	filebeatBackupMetadataKey = "filebeat/registry_backups/backup"
	registryBackupSchemaV1    = 1
	defaultRegistryBackend    = "memlog"
	backupDirName             = "registry_backups"
	backupStoreFileName       = "backup-store.db"
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

type backupStoreCloser interface {
	backend.BackupStore
	Close() error
}

func handleBackup(ctx context.Context, logger *logp.Logger, store backend.BackupStore, reg config.Registry, beatsPaths *paths.Path) error {
	if store == nil {
		logger.Debug("backup storage is not configured, skipping backup metadata persistence")
		return nil
	}

	currentState := newFilebeatStateRecord(reg)

	previousStateRaw, err := store.Get(ctx, filebeatStateKey)
	if err != nil {
		return fmt.Errorf("cannot read previous state: %w", err)
	}

	previousState := filebeatStateRecord{}
	if len(previousStateRaw) > 0 {
		if err := json.Unmarshal(previousStateRaw, &previousState); err != nil {
			return fmt.Errorf("cannot decode Filebeat backup state: %w", err)
		}
	}

	if backupNeeded(previousState, currentState) {
		logger.Info("Registry backup needed, starting.")
		defer logger.Info("Registry backup needed done.")
		registryPath := beatsPaths.Resolve(paths.Data, reg.Path)
		if err := writeBackup(logger, store, currentState, reg, registryPath); err != nil {
			return fmt.Errorf("cannot write backup: %w", err)
		}
	}

	statePayload, err := json.Marshal(currentState)
	if err != nil {
		return fmt.Errorf("failed to marshal backup state record: %w", err)
	}
	if err := store.Set(context.Background(), filebeatStateKey, statePayload); err != nil {
		return fmt.Errorf("failed to store backup state record: %w", err)
	}

	return nil
}

func openFallbackBackupStore(
	logger *logp.Logger,
	reg config.Registry,
	beatPaths *paths.Path,
) (backupStoreCloser, error) {
	path := beatPaths.Resolve(paths.Data, filepath.Join(backupDirName, backupStoreFileName))
	return bbolt.NewBackupStore(logger, path, reg.Permissions, reg.Bbolt)
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

func writeBackup(
	logger *logp.Logger,
	store backend.BackupStore,
	currentState filebeatStateRecord,
	reg config.Registry,
	registryPath string,
) error {
	if registryPath == "" {
		return nil
	}

	entries, err := os.ReadDir(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot read registry path %q: %w", registryPath, err)
	}
	if len(entries) == 0 {
		return nil
	}

	archiveRoot := filepath.Dir(registryPath)
	createdAt := time.Now().UTC()
	backupID := createdAt.Format(time.RFC3339Nano)
	backupFileName := fmt.Sprintf("registry-backup-%s.tar", backupID)
	backupDir := filepath.Join(archiveRoot, backupDirName)
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return fmt.Errorf("cannot create backup directory %q: %w", backupDir, err)
	}

	tmpBackupPath := filepath.Join(backupDir, backupFileName+".tmp")
	backupPath := filepath.Join(backupDir, backupFileName)

	logger.Infof("Creating store backup. File: %q, tmp file: %q", backupPath, tmpBackupPath)
	f, err := os.Create(tmpBackupPath)
	if err != nil {
		return fmt.Errorf("cannot create backup archive %q: %w", tmpBackupPath, err)
	}

	tw := tar.NewWriter(f)
	defer func() {
		_ = tw.Close()
		_ = f.Close()
		_ = os.Remove(tmpBackupPath)
	}()

	if err := filepath.Walk(registryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() && !info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(archiveRoot, path)
		if err != nil {
			return fmt.Errorf("cannot resolve archive path for %q: %w", path, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("cannot build tar header for %q: %w", path, err)
		}

		header.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			header.Name += "/"
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("cannot write tar directory header for %q: %w", path, err)
			}
			return nil
		}

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("cannot write tar file header for %q: %w", path, err)
		}

		in, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open registry file %q: %w", path, err)
		}
		defer in.Close()

		if _, err := io.Copy(tw, in); err != nil {
			return fmt.Errorf("cannot write registry file %q to tar archive: %w", path, err)
		}

		return nil
	}); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("cannot finalize backup archive %q: %w", tmpBackupPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("cannot close backup archive %q: %w", tmpBackupPath, err)
	}
	if err := os.Rename(tmpBackupPath, backupPath); err != nil {
		return fmt.Errorf("cannot move backup archive into place: %w", err)
	}

	backupRelativePath, err := filepath.Rel(filepath.Dir(registryPath), backupPath)
	if err != nil {
		return fmt.Errorf("cannot compute backup archive path relative to data directory: %w", err)
	}

	backupRecord := newBackupMetadataRecord(currentState, reg, createdAt, backupID, filepath.ToSlash(backupRelativePath))
	backupPayload, err := json.Marshal(backupRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal backup metadata record: %w", err)
	}
	if err := store.Set(context.Background(), filebeatBackupMetadataKey, backupPayload); err != nil {
		return fmt.Errorf("failed to store backup metadata record: %w", err)
	}

	logger.Infof("Backup successfully created")
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

func newBackupMetadataRecord(
	source filebeatStateRecord,
	reg config.Registry,
	createdAt time.Time,
	backupID string,
	backupPath string,
) backupMetadataRecord {
	return backupMetadataRecord{
		SchemaVersion: registryBackupSchemaV1,
		BackupID:      backupID,
		CreatedAt:     createdAt,
		Source:        source,
		Registry: backupRegistry{
			RelativePath: reg.Path,
			Files: []backupFiles{
				{Path: backupPath},
			},
		},
	}
}
