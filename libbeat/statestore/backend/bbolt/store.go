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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	defaultBucket     = []byte("default")
	errKeyUnknown     = errors.New("key unknown")
	errNotInitialized = errors.New("storage not initialized")
	errStopIteration  = errors.New("stop iteration")
)

// storedEntry is the on-disk format for each key-value pair.
// It wraps the actual value with a timestamp for TTL support.
type storedEntry struct {
	Value     json.RawMessage `json:"v"`
	Timestamp int64           `json:"ts"`
}

// valueDecoder implements backend.ValueDecoder for bbolt entries.
// It holds raw JSON bytes and decodes lazily on first Decode call.
type valueDecoder struct {
	raw     json.RawMessage
	decoded map[string]any
}

func (d *valueDecoder) Decode(to any) error {
	if d.decoded == nil {
		if err := json.Unmarshal(d.raw, &d.decoded); err != nil {
			return fmt.Errorf("failed to decode value: %w", err)
		}
	}
	return typeconv.Convert(to, d.decoded)
}

func (e *storedEntry) isExpired(ttl time.Duration, nowNano int64) bool {
	if ttl <= 0 {
		return false
	}
	return nowNano-e.Timestamp > ttl.Nanoseconds()
}

// store implements backend.Store backed by a single bbolt database file.
type store struct {
	log      *logp.Logger
	dbPath   string
	fileMode os.FileMode
	config   Config
	options  *bbolt.Options

	// mu guards db and closed. Read lock for normal operations (Has, Get,
	// Set, Remove, Each, cleanupExpired). Write lock for compact and Close,
	// which replace or close the db handle.
	mu     sync.RWMutex
	db     *bbolt.DB
	closed bool

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func bboltOptions(timeout time.Duration, noSync bool) *bbolt.Options {
	return &bbolt.Options{
		Timeout:        timeout,
		NoSync:         noSync,
		NoFreelistSync: true,
		FreelistType:   bbolt.FreelistMapType,
	}
}

func openStore(log *logp.Logger, dbPath string, fileMode os.FileMode, cfg Config) (*store, error) {
	resolved, err := filepath.EvalSymlinks(dbPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to resolve symlinks for %s: %w", dbPath, err)
	}
	if err == nil {
		dbPath = resolved
	}

	log.Debugf("Opening bbolt database: path=%s timeout=%v fsync=%v", dbPath, cfg.Timeout, cfg.FSync)

	options := bboltOptions(cfg.Timeout, !cfg.FSync)
	db, err := bbolt.Open(dbPath, fileMode, options)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database %s: %w", dbPath, err)
	}

	log.Debug("Ensuring default bucket exists")

	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(defaultBucket)
		return err
	}); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to create default bucket: %w", err), db.Close())
	}

	// ctx is only used to control the retention loop's shutdown
	ctx, cancel := context.WithCancel(context.Background())

	s := &store{
		log:      log,
		dbPath:   dbPath,
		fileMode: fileMode,
		config:   cfg,
		options:  options,
		db:       db,
		cancel:   cancel,
	}

	if cfg.Compaction.CleanupOnStart {
		log.Debug("Running temp file cleanup on start")
		cleanupTempFiles(log, filepath.Dir(dbPath))
	}

	if cfg.Compaction.OnStart {
		log.Debug("Running compaction on start")
		if err := s.compact(); err != nil {
			log.Warnf("Compaction on start failed: %v", err)
		}
	}

	if cfg.Retention.TTL > 0 && cfg.Retention.Interval > 0 {
		log.Debugf("Enabling retention: ttl=%v interval=%v", cfg.Retention.TTL, cfg.Retention.Interval)
		s.startRetentionLoop(ctx)
	}

	log.Debugf("Store ready: path=%s", dbPath)

	return s, nil
}

// Close stops background goroutines and closes the underlying bbolt database.
func (s *store) Close() error {
	s.cancel()
	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.log.Debugf("Closing bbolt store: path=%s", s.dbPath)

	err := s.db.Close()
	if err != nil {
		return err
	}

	s.log.Debugf("Closed bbolt store: path=%s", s.dbPath)
	return nil
}

// Has checks if the key exists in the store. Returns false for expired entries.
func (s *store) Has(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return nil
		}
		data := bucket.Get([]byte(key))
		if data == nil {
			return nil
		}
		var entry storedEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			s.log.Warnf("Failed to decode stored entry for key %q, treating as missing: %v", key, err)
			return nil
		}
		found = !entry.isExpired(s.config.Retention.TTL, time.Now().UnixNano())
		return nil
	})
	return found, err
}

// Get decodes the value for the given key into the provided value.
// Returns errKeyUnknown for missing or expired entries.
func (s *store) Get(key string, to any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return errKeyUnknown
		}

		data := bucket.Get([]byte(key))
		if data == nil {
			return errKeyUnknown
		}

		var entry storedEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("failed to decode stored entry for key %q: %w", key, err)
		}

		if entry.isExpired(s.config.Retention.TTL, time.Now().UnixNano()) {
			return errKeyUnknown
		}

		dec := &valueDecoder{raw: entry.Value}
		return dec.Decode(to)
	})
}

func encodeEntry(value any) ([]byte, error) {
	var tmp mapstr.M
	if err := typeconv.Convert(&tmp, value); err != nil {
		return nil, err
	}
	valueBytes, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}
	return json.Marshal(storedEntry{
		Value:     valueBytes,
		Timestamp: time.Now().UnixNano(),
	})
}

// Set inserts or overwrites a key-value pair in the store.
func (s *store) Set(key string, value any) error {
	entryBytes, err := encodeEntry(value)
	if err != nil {
		return fmt.Errorf("failed to encode entry for key %q: %w", key, err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return errNotInitialized
		}
		return bucket.Put([]byte(key), entryBytes)
	})
}

// Remove removes an entry from the store.
func (s *store) Remove(key string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(key))
	})
}

// Each iterates over all key-value pairs in the store, skipping expired entries.
func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ttl := s.config.Retention.TTL
	nowNano := time.Now().UnixNano()

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var entry storedEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				return fmt.Errorf("failed to decode stored entry for key %q: %w", string(k), err)
			}

			if entry.isExpired(ttl, nowNano) {
				return nil
			}

			dec := &valueDecoder{raw: entry.Value}
			cont, err := fn(string(k), dec)
			if err != nil {
				return err
			}
			if !cont {
				return errStopIteration
			}
			return nil
		})
	})

	if errors.Is(err, errStopIteration) {
		return nil
	}
	return err
}

// SetID is a no-op for the bbolt backend. The store identity is determined
// by the database file name at creation time.
func (s *store) SetID(_ string) {}
