// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"fmt"
	"os"
	"path"
	"sync"
)

// stateStore handles persistence of state markers using the filesystem
type stateStore struct {
	Dir          string // Base directory for storing state files
	sync.RWMutex        // Protects access to the state store
}

// newStateStore creates a new state store instance at the specified path
func newStateStore(path string) (*stateStore, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}
	return &stateStore{
		Dir: path,
	}, nil
}

// getStatePath builds the full file path for a given state key
func (s *stateStore) getStatePath(name string) string {
	return path.Join(s.Dir, name)
}

// Put creates a state marker file for the given key
func (s *stateStore) Put(key string) error {
	s.Lock()
	defer s.Unlock()

	filePath := s.getStatePath(key)
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating state file: %w", err)
	}
	return f.Close()
}

// Has checks if a state exists for the given key
func (s *stateStore) Has(key string) bool {
	s.RLock()
	defer s.RUnlock()

	filePath := s.getStatePath(key)
	_, err := os.Stat(filePath)
	return err == nil
}

// Remove deletes the state marker file for the given key
func (s *stateStore) Remove(key string) error {
	s.Lock()
	defer s.Unlock()

	filePath := s.getStatePath(key)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing state file: %w", err)
	}
	return nil
}

// Clear removes all state markers by deleting and recreating the state directory
func (s *stateStore) Clear() error {
	s.Lock()
	defer s.Unlock()

	if err := os.RemoveAll(s.Dir); err != nil {
		return fmt.Errorf("clearing state directory: %w", err)
	}
	return os.MkdirAll(s.Dir, 0o755)
}
