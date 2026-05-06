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

package bbolt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

var _ backend.BackupStore = (*BackupStore)(nil)

var errBackupStoreClosed = errors.New("backup store has been closed")

// BackupStore provides raw key/value storage backed by a single bbolt database.
type BackupStore struct {
	log  *logp.Logger
	path string

	mu     sync.RWMutex
	db     *bbolt.DB
	closed bool
}

// NewBackupStore creates a new bbolt-backed BackupStore.
func NewBackupStore(logger *logp.Logger, path string, fileMode os.FileMode, cfg Config) (*BackupStore, error) {
	if logger == nil {
		logger = logp.NewNopLogger()
	}
	if fileMode == 0 {
		fileMode = defaultFileMode
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid bbolt backup store config: %w", err)
	}

	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve backup store path %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), os.ModeDir|0o770); err != nil {
		return nil, fmt.Errorf("failed to create backup store directory %s: %w", filepath.Dir(resolvedPath), err)
	}

	db, err := bbolt.Open(resolvedPath, fileMode, bboltOptions(cfg.Timeout, !cfg.FSync))
	if err != nil {
		return nil, fmt.Errorf("failed to open backup store %s: %w", resolvedPath, err)
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(defaultBucket)
		return err
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize backup store bucket: %w", err)
	}

	return &BackupStore{
		log:  logger,
		path: resolvedPath,
		db:   db,
	}, nil
}

// Get returns the raw value stored for key, or nil if the key does not exist.
func (s *BackupStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, errBackupStoreClosed
	}

	var value []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return nil
		}

		raw := bucket.Get([]byte(key))
		if raw == nil {
			return nil
		}

		value = append([]byte(nil), raw...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return value, nil
}

// Set stores raw bytes for key.
func (s *BackupStore) Set(ctx context.Context, key string, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return errBackupStoreClosed
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return errNotInitialized
		}

		return bucket.Put([]byte(key), append([]byte(nil), value...))
	})
}

// Delete removes the value stored for key.
func (s *BackupStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return errBackupStoreClosed
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return nil
		}

		return bucket.Delete([]byte(key))
	})
}

// Close closes the underlying bbolt database.
func (s *BackupStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.log.Debugf("Closing bbolt backup store: path=%s", s.path)
	return s.db.Close()
}
