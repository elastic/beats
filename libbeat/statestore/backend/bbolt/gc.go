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
//
// This file was contributed to by generative AI

package bbolt

import (
	"encoding/json"
	"fmt"
	"time"

	"slices"

	"go.etcd.io/bbolt"
)

// collectGarbage performs a full scan of the metadata bucket and deletes
// expired entries from both metadata and data buckets.
//
// Phase 1/2: full scan. Phase 3 will introduce incremental GC.
func (s *store) collectGarbage() error {
	if s.settings.DiskTTL <= 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.requireOpen(); err != nil {
		return err
	}

	start := s.now()
	nowNanos := start.UnixNano()
	ttlNanos := s.settings.DiskTTL.Nanoseconds()

	var (
		scanned int
		deleted int
	)

	err := s.db.Update(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(bucketMetadata)
		if metaBucket == nil {
			return nil
		}
		dataBucket := tx.Bucket(bucketData)

		c := metaBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			scanned++

			var meta metadata
			if err := json.Unmarshal(v, &meta); err != nil {
				s.logger.Warnf("Failed to unmarshal metadata for key %s: %v", string(k), err)
				continue
			}

			// If there's no access timestamp (0), consider it active (conservative).
			if meta.LastAccess == 0 {
				continue
			}

			age := nowNanos - meta.LastAccess
			if age <= ttlNanos {
				continue
			}

			key := slices.Clone(k)
			if dataBucket != nil {
				if err := dataBucket.Delete(key); err != nil {
					return fmt.Errorf("delete data key %q: %w", string(key), err)
				}
			}
			if err := c.Delete(); err != nil {
				return fmt.Errorf("delete metadata key %q: %w", string(key), err)
			}
			deleted++
		}

		return nil
	})
	if err != nil {
		return err
	}

	duration := s.now().Sub(start)
	s.logger.Infow("GC completed",
		"duration_ms", duration.Milliseconds(),
		"scanned", scanned,
		"deleted", deleted,
		"gc_type", "full_scan",
	)

	if duration > 10*time.Second {
		s.logger.Warnf("GC took %v to scan %d entries. Consider enabling incremental GC (Phase 3)", duration, scanned)
	}

	return nil
}
