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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	bolt "go.etcd.io/bbolt"
)

const (
	oneMiB       = 1 << 20
	tempDbPrefix = "tempdb"
)

// compact performs database compaction by copying data to a temporary database
// and replacing the original. This reclaims unused disk space.
func (s *store) compact() error {
	compactionDir := filepath.Dir(s.dbPath)

	file, err := os.CreateTemp(compactionDir, tempDbPrefix)
	if err != nil {
		return fmt.Errorf("failed to create temp file for compaction: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}

	defer func() {
		if removeErr := os.Remove(file.Name()); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.log.Errorf("Failed to remove temporary compaction file: %v", removeErr)
		}
	}()

	s.compactionMu.Lock()
	defer s.compactionMu.Unlock()

	if s.closed {
		s.log.Debug("Skipping compaction, store is already closed")
		return nil
	}

	s.log.Debugf("Starting compaction: path=%s temp=%s", s.dbPath, file.Name())

	compactedDB, err := bolt.Open(file.Name(), s.fileMode, s.options)
	if err != nil {
		return fmt.Errorf("failed to open temp db for compaction: %w", err)
	}

	compactionStart := time.Now()

	if err := bolt.Compact(compactedDB, s.db, s.config.Compaction.MaxTransactionSize); err != nil {
		compactedDB.Close()
		return fmt.Errorf("compaction failed: %w", err)
	}

	dbPath := s.db.Path()
	compactedPath := compactedDB.Path()

	s.db.Close()
	compactedDB.Close()

	moveErr := moveFileWithFallback(compactedPath, dbPath)

	s.db, err = bolt.Open(dbPath, s.fileMode, s.options)
	if err != nil {
		return errors.Join(
			fmt.Errorf("failed to reopen db after compaction: %w", err),
			moveErr,
		)
	}

	if moveErr != nil {
		var pathErr *os.PathError
		if errors.As(moveErr, &pathErr) && pathErr.Op == "remove" {
			s.log.Warnf("Compaction succeeded but failed to remove temp file: %v", moveErr)
		} else {
			return fmt.Errorf("failed to replace db with compacted file: %w", moveErr)
		}
	}

	s.log.Debugf("Finished compaction in %v", time.Since(compactionStart))
	return nil
}

// runLoop starts a background goroutine that calls fn on every tick of interval
// and stops when the store's stopCh is closed.
func (s *store) runLoop(name string, interval time.Duration, fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fn()
			case <-s.stopCh:
				s.log.Debugf("Shutting down %s loop", name)
				return
			}
		}
	}()
}

// shouldCompact checks whether the conditions for online rebound compaction
// are met based on the configured thresholds.
func (s *store) shouldCompact() bool {
	if !s.config.Compaction.OnRebound {
		return false
	}

	s.compactionMu.RLock()
	defer s.compactionMu.RUnlock()

	var totalSize int64
	err := s.db.View(func(tx *bolt.Tx) error {
		totalSize = tx.Size()
		return nil
	})
	if err != nil {
		s.log.Errorf("Failed to get db size: %v", err)
		return false
	}

	stats := s.db.Stats()
	dataSize := totalSize - int64(stats.FreeAlloc)

	if dataSize > s.config.Compaction.ReboundTriggerThresholdMiB*oneMiB ||
		totalSize < s.config.Compaction.ReboundNeededThresholdMiB*oneMiB {
		return false
	}

	s.log.Debugf("Rebound compaction triggered: totalSize=%d dataSize=%d", totalSize, dataSize)
	return true
}

// cleanupExpired removes entries from the store whose timestamp is older
// than the configured TTL. Deletes are batched by MaxTransactionSize to
// limit memory usage and transaction duration. A cursor is used to resume
// scanning from the last position between batches.
func (s *store) cleanupExpired() error {
	if s.config.TTL <= 0 {
		return nil
	}

	s.log.Debug("Running TTL cleanup")

	s.compactionMu.RLock()
	defer s.compactionMu.RUnlock()

	now := time.Now().UnixNano()
	ttlNanos := s.config.TTL.Nanoseconds()
	batchSize := s.config.Compaction.MaxTransactionSize
	var totalRemoved int
	var seekKey []byte

	for {
		var keysToDelete [][]byte
		var scannedAll bool

		err := s.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(defaultBucket)
			if bucket == nil {
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
				if err := bucket.Delete(key); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		totalRemoved += len(keysToDelete)

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

// moveFileWithFallback attempts os.Rename first. If it fails due to a
// cross-device link error (EXDEV), it falls back to a streaming
// copy-and-remove that avoids loading the entire file into memory.
func moveFileWithFallback(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	if _, err := io.Copy(destFile, srcFile); err != nil {
		destFile.Close()
		return err
	}

	if err := destFile.Close(); err != nil {
		return err
	}

	return os.Remove(src)
}
