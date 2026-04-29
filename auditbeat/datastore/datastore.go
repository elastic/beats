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
	"io"

	bolt "go.etcd.io/bbolt"

	"github.com/elastic/elastic-agent-libs/paths"
)

const (
	dbFileName = "beat.db"
	dbFileMode = 0o600
)

// OpenBucket returns a Bucket that stores data in {path.data}/beat.db.
// The returned Bucket must be closed when no longer needed; the underlying
// database is closed when the last bucket for a given path is closed.
func OpenBucket(name string, p *paths.Path) (Bucket, error) {
	return defaultRegistry.openBucket(name, p, nil)
}

// OpenBucketWithMigration is like OpenBucket but runs migrate in a
// read-write transaction before the named bucket is ensured to exist.
// migrate must be idempotent: it will run on every call, including after
// process restarts.
func OpenBucketWithMigration(name string, p *paths.Path, migrate func(tx *bolt.Tx) error) (Bucket, error) {
	return defaultRegistry.openBucket(name, p, migrate)
}

// Bucket is a key-value bucket within the datastore.
type Bucket interface {
	io.Closer
	Load(key string, f func(blob []byte) error) error
	Store(key string, blob []byte) error
	Delete(key string) error // Delete removes a key from the bucket. If the key does not exist then nothing is done and a nil error is returned.
	DeleteBucket() error     // Deletes and closes the bucket.
}

// BoltBucket is a Bucket that exposes some Bolt specific APIs.
type BoltBucket interface {
	Bucket
	View(func(tx *bolt.Bucket) error) error
	Update(func(tx *bolt.Bucket) error) error
}

// boltBucket implements Bucket and BoltBucket. It holds exactly one
// reference on db that Close releases.
type boltBucket struct {
	db   *boltDB
	name []byte
}

func (b *boltBucket) Load(key string, f func(blob []byte) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.name)
		data := bucket.Get([]byte(key))
		if data == nil {
			return nil
		}
		return f(data)
	})
}

func (b *boltBucket) Store(key string, blob []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.name)
		return bucket.Put([]byte(key), blob)
	})
}

func (b *boltBucket) ForEach(f func(key string, blob []byte) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.name)
		return bucket.ForEach(func(k, v []byte) error {
			return f(string(k), v)
		})
	})
}

func (b *boltBucket) Delete(key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.name)
		return bucket.Delete([]byte(key))
	})
}

func (b *boltBucket) DeleteBucket() error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(b.name)
	})
	// Always release the reference, even if the delete failed, so we
	// don't leak the database open forever.
	if closeErr := b.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (b *boltBucket) View(f func(*bolt.Bucket) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		return f(tx.Bucket(b.name))
	})
}

func (b *boltBucket) Update(f func(*bolt.Bucket) error) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		return f(tx.Bucket(b.name))
	})
}

func (b *boltBucket) Close() error {
	return defaultRegistry.releaseBucket(b.db)
}
