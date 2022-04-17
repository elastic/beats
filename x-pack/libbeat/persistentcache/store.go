// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	badger "github.com/dgraph-io/badger/v3"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

// Store is a store for a persistent cache. It can be shared between consumers.
type Store struct {
	logger *logp.Logger
	name   string

	gcQuit chan struct{}
	db     *badger.DB
}

// newStore opens a store persisted on the specified directory, with the specified name.
// If succeeds, returned store must be closed.
func newStore(logger *logp.Logger, dir, name string) (*Store, error) {
	dbPath := filepath.Join(dir, name)
	err := os.MkdirAll(dbPath, 0750)
	if err != nil {
		return nil, fmt.Errorf("creating directory for cache store: %w", err)
	}

	// Opinionated options for the use of badger as a store for metadata caches in Beats.
	options := badger.DefaultOptions(dbPath)
	options.Logger = badgerLogger{logger.Named("badger")}
	options.SyncWrites = false

	db, err := badger.Open(options)
	if err != nil {
		return nil, fmt.Errorf("opening database for cache store: %w", err)
	}

	store := Store{
		db:     db,
		logger: logger,
		name:   name,
		gcQuit: make(chan struct{}),
	}
	go store.runGC(gcPeriod)
	return &store, nil
}

// Close closes the store.
func (s *Store) Close() error {
	s.stopGC()
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("closing database of cache store: %w", err)
	}
	return nil
}

// Set sets a value with a ttl in the store. If ttl is zero, it is ignored.
func (s *Store) Set(k, v []byte, ttl time.Duration) error {
	entry := badger.Entry{
		Key:   []byte(k),
		Value: v,
	}
	if ttl > 0 {
		entry.WithTTL(ttl)
	}
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(&entry)
	})
	if err != nil {
		return fmt.Errorf("setting value in cache store: %w", err)
	}
	return err
}

// Get gets a value from the store.
func (s *Store) Get(k []byte) ([]byte, error) {
	var result []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(k)
		if err != nil {
			return err
		}
		result, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getting value from cache store: %w", err)
	}
	return result, nil
}

// runGC starts garbage collection in the store.
func (s *Store) runGC(period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var err error
			count := 0
			for err == nil {
				err = s.db.RunValueLogGC(0.5)
				count++
			}
			s.logger.Debugf("Result of garbage collector after running %d times: %s", count, err)
		case <-s.gcQuit:
			return
		}
	}

}

// stopGC stops garbage collection in the store.
func (s *Store) stopGC() {
	close(s.gcQuit)
}

// badgerLogger is an adapter between a logp logger and the loggers expected by badger.
type badgerLogger struct {
	*logp.Logger
}

// Warningf logs a message at the warning level.
func (l badgerLogger) Warningf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}
