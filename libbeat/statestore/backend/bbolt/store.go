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
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"slices"

	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	dataBucketName     = "data"
	metadataBucketName = "metadata"
)

var (
	bucketData     = []byte(dataBucketName)
	bucketMetadata = []byte(metadataBucketName)
)

type store struct {
	db     *bbolt.DB
	logger *logp.Logger
	now    func() time.Time

	mu     sync.RWMutex
	closed bool

	name     string
	path     string
	settings Settings
}

type metadata struct {
	LastAccess int64 `json:"last_access"`
	LastChange int64 `json:"last_change"`
}

type entry struct {
	value any
}

func (e entry) Decode(to any) error {
	return typeconv.Convert(to, e.value)
}

// jsonDecoder implements backend.ValueDecoder
type jsonDecoder struct {
	raw []byte
}

func (d jsonDecoder) Decode(to any) error {
	var tmp any
	if err := json.Unmarshal(d.raw, &tmp); err != nil {
		return err
	}
	return typeconv.Convert(to, tmp)
}

func openStore(logger *logp.Logger, path string, settings Settings) (*store, error) {
	if logger == nil {
		return nil, fmt.Errorf("open bbolt store: logger is nil")
	}
	if path == "" {
		return nil, fmt.Errorf("open bbolt store: path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create bbolt store directory %q: %w", dir, err)
	}

	opts := &bbolt.Options{
		Timeout:        settings.Timeout,
		NoGrowSync:     settings.NoGrowSync,
		NoFreelistSync: settings.NoFreelistSync,
	}

	db, err := bbolt.Open(path, settings.FileMode, opts)
	if err != nil {
		return nil, fmt.Errorf("open bbolt DB %q: %w", path, err)
	}

	if err := ensureFilePermissions(path, settings.FileMode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ensure bbolt DB permissions %q: %w", path, err)
	}

	s := &store{
		db:       db,
		logger:   logger,
		now:      time.Now,
		name:     filepath.Base(path),
		path:     path,
		settings: settings,
	}

	if err := s.initBuckets(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *store) initBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketData); err != nil {
			return fmt.Errorf("create bucket %q: %w", dataBucketName, err)
		}
		if _, err := tx.CreateBucketIfNotExists(bucketMetadata); err != nil {
			return fmt.Errorf("create bucket %q: %w", metadataBucketName, err)
		}
		return nil
	})
}

func (s *store) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *store) requireOpen() error {
	if s.closed {
		return errStoreClosed
	}
	return nil
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.db.Close()
}

func (s *store) Has(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.requireOpen(); err != nil {
		return false, err
	}

	var exists bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketData)
		if b == nil {
			exists = false
			return nil
		}
		exists = b.Get([]byte(key)) != nil
		return nil
	})
	return exists, err
}

func (s *store) Get(key string, to any) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.requireOpen(); err != nil {
		return err
	}

	var raw []byte
	err := s.db.Update(func(tx *bbolt.Tx) error {
		data := tx.Bucket(bucketData)
		if data == nil {
			return errKeyUnknown
		}
		v := data.Get([]byte(key))
		if v == nil {
			return errKeyUnknown
		}
		raw = slices.Clone(v)
		return s.updateAccessTime(tx, key)
	})
	if err != nil {
		return err
	}

	var tmp any
	if err := json.Unmarshal(raw, &tmp); err != nil {
		return err
	}
	return typeconv.Convert(to, tmp)
}

func (s *store) Set(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.requireOpen(); err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketData)
		if b == nil {
			return fmt.Errorf("data bucket missing")
		}
		if err := b.Put([]byte(key), data); err != nil {
			return err
		}
		return s.updateMetadata(tx, key, true)
	})
}

func (s *store) Remove(key string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.requireOpen(); err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		if b := tx.Bucket(bucketData); b != nil {
			_ = b.Delete([]byte(key))
		}
		if b := tx.Bucket(bucketMetadata); b != nil {
			_ = b.Delete([]byte(key))
		}
		return nil
	})
}

func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.requireOpen(); err != nil {
		return err
	}

	return s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketData)
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)
			raw := slices.Clone(v)

			// TODO(Tiago): We might not need the clone above because
			// we sto pusing the values returned by next are valid until
			// the end of the transaction, and we only use the values to
			// decode/convert them into the final type.
			cont, err := fn(key, jsonDecoder{raw: raw})
			if !cont || err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *store) updateAccessTime(tx *bbolt.Tx, key string) error {
	bucket := tx.Bucket(bucketMetadata)
	if bucket == nil {
		return nil
	}

	nowNanos := s.now().UnixNano()

	var meta metadata
	if v := bucket.Get([]byte(key)); v != nil {
		if err := json.Unmarshal(v, &meta); err != nil {
			// Best-effort: keep scanning; rewrite metadata from scratch.
			meta = metadata{}
		}
	}
	meta.LastAccess = nowNanos

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}

func (s *store) updateMetadata(tx *bbolt.Tx, key string, changeTime bool) error {
	bucket := tx.Bucket(bucketMetadata)
	if bucket == nil {
		return nil
	}

	nowNanos := s.now().UnixNano()

	meta := metadata{
		LastAccess: nowNanos,
	}

	if changeTime {
		meta.LastChange = nowNanos
	} else if v := bucket.Get([]byte(key)); v != nil {
		var existing metadata
		if err := json.Unmarshal(v, &existing); err == nil {
			meta.LastChange = existing.LastChange
		}
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}

func (s *store) SetID(_ string) {
	// NOOP
}

// DB returns the underlying bbolt database for read-only access.
// Returns nil if store is closed.
func (s *store) DB() *bbolt.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil
	}
	return s.db
}

func ensureFilePermissions(path string, wantPerm os.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	f, err := os.OpenFile(path, os.O_RDWR, wantPerm)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	wantPerm = wantPerm & os.ModePerm
	perm := fi.Mode() & os.ModePerm
	if wantPerm == perm {
		return nil
	}

	return f.Chmod((fi.Mode() &^ os.ModePerm) | wantPerm)
}
