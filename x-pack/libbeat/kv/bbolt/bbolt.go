// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package bbolt

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	defaultDbPath     = "beat_cache.db"
	defaultBucketName = "kv"
	defaultDbFileMode = 0o600
)

// BboltValue - value type used for storage in bolt DB.
type BboltValue struct {
	RawValue []byte `json:"rawValue"`

	// ExpireAt - Unix timestamp (in nanoseconds) of time when value expires.
	ExpireAt int64 `json:"expireAt"`

	// TTL - Time To Live used for value. If 0 then the value doesn't expire
	TTL time.Duration `json:"ttl"`
}

type Option func(bbolt *Bbolt)

type Bbolt struct {
	dbPath     string
	dbFileMode os.FileMode
	bucketName string

	db *bolt.DB
}

// New creates and returns instance of bolt key-value cache implementation
func New(options ...Option) *Bbolt {
	b := &Bbolt{
		dbPath:     defaultDbPath,
		dbFileMode: defaultDbFileMode,
		bucketName: defaultBucketName,
	}
	for _, opt := range options {
		opt(b)
	}

	return b
}

func WithDbPath(path string) Option {
	return func(b *Bbolt) {
		b.dbPath = path
	}
}

func WithDbFileMode(mode os.FileMode) Option {
	return func(b *Bbolt) {
		b.dbFileMode = mode
	}
}

func WithBucketName(name string) Option {
	return func(b *Bbolt) {
		b.bucketName = name
	}
}

// Connect creates directories of a given path for bbolt DB file (if directories not already exist), creates DB file with given file permissions, creates bucket to store cache data.
func (b *Bbolt) Connect() error {
	var err error

	dbDir := path.Dir(b.dbPath)
	err = os.MkdirAll(dbDir, b.dbFileMode)
	if err != nil {
		return fmt.Errorf("bbolt: creation of the directory for DB failed: %w", err)
	}

	b.db, err = openDbFile(b.dbPath, b.dbFileMode)
	if err != nil {
		return fmt.Errorf("bbolt: openDbFile error: %w", err)
	}
	err = b.ensureBucketExists()
	if err != nil {
		return fmt.Errorf("bbolt: bucket opening error: %w", err)
	}
	return nil
}

// Get fetches value by key from bolt DB (returns nil if key is not present or expired)
func (b *Bbolt) Get(key []byte) (data []byte, err error) {
	// we need writable transaction here in order to delete expired keys
	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucketName))

		jsonVal := bucket.Get(key)
		if jsonVal == nil { // no value in store
			return nil
		}

		var val BboltValue
		if err := json.Unmarshal(jsonVal, &val); err != nil {
			return err
		}
		if val.TTL > 0 && val.ExpireAt <= time.Now().UnixNano() { // value expired
			return bucket.Delete(key)
		}
		data = val.RawValue
		return nil
	})
	return data, err
}

// Set stores a key-value pair in the database. If TTL is 0, the key does not expire.
func (b *Bbolt) Set(key []byte, value []byte, ttl time.Duration) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(b.bucketName))

		bboltValEncoded, err := getMarshalledBboltValue(value, ttl)
		if err != nil {
			return err
		}
		err = bucket.Put(key, bboltValEncoded)
		if err != nil {
			return err
		}

		return nil
	})
}

// Close closes the database.
func (b *Bbolt) Close() error {
	return b.db.Close()
}

// ensureBucketExists - creates bolt bucket if it doesn't already exist.
func (b *Bbolt) ensureBucketExists() error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(b.bucketName))
		return err
	})
	return err
}

// openDbFile opens bolt DB file and returns *bolt.DB instance
func openDbFile(path string, mode os.FileMode) (*bolt.DB, error) {
	db, err := bolt.Open(path, mode, nil)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// getMarshalledBboltValue returns json encoded BboltValue constructed from raw value and TTL.
func getMarshalledBboltValue(value []byte, ttl time.Duration) ([]byte, error) {
	return json.Marshal(newBboltValue(value, ttl))
}

// newBboltValue creates BboltValue from raw value and TTL
func newBboltValue(value []byte, ttl time.Duration) BboltValue {
	var expireAt int64
	if ttl > 0 {
		expireAt = time.Now().UnixNano() + ttl.Nanoseconds()
	}
	return BboltValue{
		RawValue: value,
		ExpireAt: expireAt,
		TTL:      ttl,
	}
}
