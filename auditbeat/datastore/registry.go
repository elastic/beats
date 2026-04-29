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

package datastore

import (
	"errors"
	"fmt"
	"sync"

	bolt "go.etcd.io/bbolt"

	"github.com/elastic/elastic-agent-libs/paths"
)

// registry tracks one refcounted bolt database per resolved data-directory path.
type registry struct {
	mu  sync.Mutex
	dbs map[string]*boltDB
}

// boltDB is the per-path state owned by a registry. The path and refCount
// fields are protected by registry.mu; the embedded *bolt.DB is itself
// goroutine-safe and can be used without holding registry.mu as long as
// the caller still holds an outstanding reference (i.e. has not yet called
// Close on its boltBucket).
type boltDB struct {
	*bolt.DB
	path     string
	refCount int
}

var defaultRegistry = &registry{dbs: map[string]*boltDB{}}

// openBucket acquires the database for p, optionally runs migrate, ensures the bucket exists, and returns it.
func (r *registry) openBucket(name string, p *paths.Path, migrate func(tx *bolt.Tx) error) (Bucket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	path := p.Resolve(paths.Data, dbFileName)
	db, err := r.acquireLocked(path)
	if err != nil {
		return nil, err
	}

	// Run migrate (if any) and ensure the bucket exists in a single
	// transaction so a failed bucket creation also rolls back the migration.
	nameBytes := []byte(name)
	err = db.Update(func(tx *bolt.Tx) error {
		if migrate != nil {
			if err := migrate(tx); err != nil {
				return fmt.Errorf("datastore migration failed: %w", err)
			}
		}
		if _, err := tx.CreateBucketIfNotExists(nameBytes); err != nil {
			return fmt.Errorf("failed to create bucket %q: %w", name, err)
		}
		return nil
	})
	if err != nil {
		if releaseErr := r.releaseLocked(db); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("releasing reference after error: %w", releaseErr))
		}
		return nil, err
	}

	return &boltBucket{db: db, name: nameBytes}, nil
}

func (r *registry) releaseBucket(db *boltDB) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.releaseLocked(db)
}

func (r *registry) acquireLocked(path string) (*boltDB, error) {
	if db, ok := r.dbs[path]; ok {
		db.refCount++
		return db, nil
	}
	opened, err := bolt.Open(path, dbFileMode, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open datastore at %q: %w", path, err)
	}
	db := &boltDB{DB: opened, path: path, refCount: 1}
	r.dbs[path] = db
	return db, nil
}

func (r *registry) releaseLocked(db *boltDB) error {
	if db.refCount == 0 {
		return errors.New("datastore: release called on a bucket with no outstanding references")
	}
	db.refCount--
	if db.refCount > 0 {
		return nil
	}
	delete(r.dbs, db.path)
	return db.Close()
}
