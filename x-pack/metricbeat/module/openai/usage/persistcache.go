// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// stateManager handles the storage and retrieval of state data with key hashing and caching capabilities
type stateManager struct {
	mu        sync.RWMutex
	store     *stateStore
	keyPrefix string
	hashCache sync.Map
}

// stateStore handles persistence of key-value pairs using the filesystem
type stateStore struct {
	Dir string // Base directory for storing state files
	mu  sync.RWMutex
}

// newStateManager creates a new state manager instance with the given storage path
func newStateManager(storePath string) (*stateManager, error) {
	if strings.TrimSpace(storePath) == "" {
		return nil, errors.New("empty path provided")
	}

	store, err := newStateStore(storePath)
	if err != nil {
		return nil, fmt.Errorf("create state store: %w", err)
	}

	return &stateManager{
		mu:        sync.RWMutex{},
		store:     store,
		keyPrefix: "state_",
		hashCache: sync.Map{},
	}, nil
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

// Put stores a value in a file named by the key
func (s *stateStore) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.getStatePath(key)

	// In case the file already exists, file is truncated.
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating state file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(value)
	if err != nil {
		return fmt.Errorf("writing value to state file: %w", err)
	}

	if err = f.Sync(); err != nil {
		return fmt.Errorf("syncing state file: %w", err)
	}

	return nil
}

// Get retrieves the value stored in the file named by the key
func (s *stateStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getStatePath(key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading state file: %w", err)
	}
	return string(data), nil
}

// Has checks if a state exists for the given key
func (s *stateStore) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getStatePath(key)
	_, err := os.Stat(filePath)
	return err == nil
}

// Remove deletes the state file for the given key
func (s *stateStore) Remove(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.getStatePath(key)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing state file: %w", err)
	}
	return nil
}

// Clear removes all state files by deleting and recreating the state directory
func (s *stateStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.RemoveAll(s.Dir); err != nil {
		return fmt.Errorf("clearing state directory: %w", err)
	}
	return os.MkdirAll(s.Dir, 0o755)
}

// GetLastProcessedDate retrieves and parses the last processed date for a given API key
func (s *stateManager) GetLastProcessedDate(apiKey string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateKey := s.GetStateKey(apiKey)

	if !s.store.Has(stateKey) {
		return time.Time{}, ErrNoState
	}

	dateStr, err := s.store.Get(stateKey)
	if err != nil {
		return time.Time{}, fmt.Errorf("get state: %w", err)
	}

	return time.Parse(dateFormatForStateStore, dateStr)
}

// SaveState saves the last processed date for a given API key
func (s *stateManager) SaveState(apiKey, dateStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stateKey := s.GetStateKey(apiKey)
	return s.store.Put(stateKey, dateStr)
}

// hashKey generates and caches a SHA-256 hash of the provided API key
func (s *stateManager) hashKey(apiKey string) string {
	// Check cache first to avoid recomputing hashes
	if hashedKey, ok := s.hashCache.Load(apiKey); ok {
		return hashedKey.(string)
	}

	// Generate SHA-256 hash and hex encode for safe filename usage
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(apiKey))
	hashedKey := hex.EncodeToString(hasher.Sum(nil))

	// Cache the computed hash for future lookups
	s.hashCache.Store(apiKey, hashedKey)
	return hashedKey
}

// GetStateKey generates a unique state key for a given API key
func (s *stateManager) GetStateKey(apiKey string) string {
	return s.keyPrefix + s.hashKey(apiKey)
}
