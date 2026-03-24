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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	bolt "go.etcd.io/bbolt"
)

const tempDbPrefix = "tempdb"

// compact performs database compaction by copying data to a temporary database
// and replacing the original. This reclaims unused disk space.
func (s *store) compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		s.log.Debug("Skipping compaction, store is already closed")
		return nil
	}

	compactionDir := filepath.Dir(s.dbPath)

	file, err := os.CreateTemp(compactionDir, tempDbPrefix)
	if err != nil {
		return fmt.Errorf("failed to create temp file for compaction: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(file.Name()); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.log.Errorf("Failed to remove temporary compaction file: %v", removeErr)
		}
	}()
	if err := file.Close(); err != nil {
		return err
	}

	compactionStart := time.Now()
	s.log.Debugf("Starting compaction: path=%s temp=%s", s.dbPath, file.Name())
	err = s.compactTo(file.Name())
	if err != nil {
		return err
	}
	// We have to close the db before renaming on Windows.
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close db for compaction: %w", err)
	}

	// Ensure s.db is reopened from s.dbPath if anything below fails.
	// On success, reopened is set to true and the defer is a no-op.
	var reopened bool
	defer func() {
		if reopened {
			return
		}
		newDB, err := bolt.Open(s.dbPath, s.fileMode, s.options)
		if err != nil {
			s.log.Errorf("Failed to reopen db after compaction failure: %v", err)
			return
		}
		s.db = newDB
	}()

	if err := os.Chmod(file.Name(), s.fileMode); err != nil {
		return fmt.Errorf("failed to set permissions on compacted db: %w", err)
	}

	if err := os.Rename(file.Name(), s.dbPath); err != nil {
		return fmt.Errorf("failed to replace db with compacted file: %w", err)
	}

	newDB, err := bolt.Open(s.dbPath, s.fileMode, s.options)
	if err != nil {
		return fmt.Errorf("failed to reopen db after compaction: %w", err)
	}
	s.db = newDB
	reopened = true

	s.log.Debugf("Finished compaction in %v", time.Since(compactionStart))
	return nil
}

// compactTo copies all data from the current database into a new database at
// the given path using bolt.Compact. The new database is closed before returning.
func (s *store) compactTo(path string) error {
	compactedDB, err := bolt.Open(path, s.fileMode, s.options)
	if err != nil {
		return fmt.Errorf("failed to open temp db for compaction: %w", err)
	}
	defer compactedDB.Close()

	if err := bolt.Compact(compactedDB, s.db, s.config.Compaction.MaxTransactionSize); err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	return nil
}

// startRetentionLoop runs a background goroutine that periodically removes
// expired entries. It stops when ctx is cancelled.
func (s *store) startRetentionLoop(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(s.config.Retention.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.cleanupExpired(); err != nil {
					s.log.Errorf("TTL cleanup failed: %v", err)
				}
			case <-ctx.Done():
				s.log.Debug("Shutting down retention loop")
				return
			}
		}
	}()
}

// cleanupExpired removes entries from the store whose timestamp is older
// than the configured TTL. Deletes are batched by MaxTransactionSize to
// limit memory usage and transaction duration. A cursor is used to resume
// scanning from the last position between batches.
func (s *store) cleanupExpired() error {
	if s.config.Retention.TTL <= 0 {
		return nil
	}

	s.log.Debug("Running TTL cleanup")

	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UnixNano()
	ttlNanos := s.config.Retention.TTL.Nanoseconds()
	batchSize := s.config.Compaction.MaxTransactionSize
	var totalRemoved int
	var seekKey []byte

	for {
		var keysToDelete [][]byte
		var scannedAll bool

		err := s.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(defaultBucket)
			if bucket == nil {
				// the bucket does not exist, so nothing to delete
				scannedAll = true
				return nil
			}
			c := bucket.Cursor()

			var k, v []byte
			if seekKey == nil {
				k, v = c.First()
			} else {
				k, v = c.Seek(seekKey)
				if k != nil && bytes.Equal(k, seekKey) {
					k, v = c.Next()
				}
			}

			for ; k != nil; k, v = c.Next() {
				if batchSize > 0 && int64(len(keysToDelete)) >= batchSize {
					return nil
				}

				seekKey = append(seekKey[:0], k...)

				var entry storedEntry
				if err := json.Unmarshal(v, &entry); err != nil {
					s.log.Warnf("Failed to decode entry for key %q during TTL cleanup, skipping: %v", string(k), err)
					continue
				}
				if now-entry.Timestamp > ttlNanos {
					keyCopy := make([]byte, len(k))
					copy(keyCopy, k)
					keysToDelete = append(keysToDelete, keyCopy)
				}
			}
			scannedAll = true
			return nil
		})
		if err != nil {
			return err
		}

		if len(keysToDelete) == 0 {
			break
		}

		err = s.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(defaultBucket)
			if bucket == nil {
				return nil
			}
			for _, key := range keysToDelete {
				v := bucket.Get(key)
				if v == nil {
					continue
				}
				var entry storedEntry
				if err := json.Unmarshal(v, &entry); err != nil {
					s.log.Warnf("Failed to decode entry for key %q during TTL re-check, skipping: %v", string(key), err)
					continue
				}
				if now-entry.Timestamp <= ttlNanos {
					continue
				}
				if err := bucket.Delete(key); err != nil {
					return err
				}
				totalRemoved++
			}
			return nil
		})
		if err != nil {
			return err
		}

		if scannedAll {
			break
		}
	}

	if totalRemoved > 0 {
		s.log.Debugf("TTL cleanup removed %d expired entries", totalRemoved)
	} else {
		s.log.Debug("TTL cleanup found no expired entries")
	}
	return nil
}

// cleanupTempFiles removes leftover temporary compaction files from a
// previous process that may have been killed during compaction.
func cleanupTempFiles(log *logp.Logger, dir string) {
	pattern := filepath.Join(dir, tempDbPrefix+"*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Warnf("Failed to list temp files for cleanup: %v", err)
		return
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			log.Warnf("Failed to remove temp file %s: %v", match, err)
		} else {
			log.Debugf("Cleaned up temp file: %s", match)
		}
	}
}
