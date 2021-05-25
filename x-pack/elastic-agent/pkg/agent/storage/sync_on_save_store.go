// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"io"
	"os"
	"runtime"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

// SyncOnSaveStore syncs paths after successful write.
type SyncOnSaveStore struct {
	enabled  bool
	syncPath string
	wrapped  Store
}

// NewSyncStore creates always syncing store.
func NewSyncStore(wrappedStore Store, syncPath string) *SyncOnSaveStore {
	return &SyncOnSaveStore{
		enabled:  true,
		syncPath: syncPath,
		wrapped:  wrappedStore,
	}
}

// NewWindowsSyncOnSaveStore creates windows syncing store.
func NewWindowsSyncOnSaveStore(wrappedStore Store, syncPath string) *SyncOnSaveStore {
	// TODO: think of windows 7 only syncing store as this is the slowest
	return &SyncOnSaveStore{
		enabled:  runtime.GOOS == "windows",
		syncPath: syncPath,
		wrapped:  wrappedStore,
	}
}

// Save accepts a persistedConfig and saved it to a target file, to do so we will
// make a temporary files if the write is successful we are replacing the target file with the
// original content.
func (s *SyncOnSaveStore) Save(in io.Reader) error {
	if err := s.wrapped.Save(in); err != nil {
		return err
	}

	if !s.enabled {
		return nil
	}

	f, err := os.OpenFile(s.syncPath, os.O_RDWR, 0777)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	defer f.Close()
	return f.Sync()
}

// Load return a io.ReadCloser for the target file.
func (s *SyncOnSaveStore) Load() (io.ReadCloser, error) {
	type loader interface {
		Load() (io.ReadCloser, error)
	}

	if loader, ok := s.wrapped.(loader); ok {
		return loader.Load()
	}

	return nil, errors.New("load is not supported for this store")
}
