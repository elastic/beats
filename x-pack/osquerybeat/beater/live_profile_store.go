// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/elastic/elastic-agent-libs/logp"
)

const liveProfileFilePrefix = "live-profile-"

type liveProfileStore struct {
	log         *logp.Logger
	dir         string
	maxProfiles int

	mx    sync.Mutex
	cache *lru.Cache[string, string]
}

func newLiveProfileStore(log *logp.Logger, dir string, maxProfiles int) (*liveProfileStore, error) {
	if maxProfiles <= 0 {
		return nil, fmt.Errorf("max profiles must be positive")
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create live profile dir: %w", err)
	}

	store := &liveProfileStore{
		log:         log,
		dir:         dir,
		maxProfiles: maxProfiles,
	}

	cache, err := lru.NewWithEvict[string, string](maxProfiles, func(_ string, filename string) {
		if filename == "" {
			return
		}
		if err := os.Remove(filename); err != nil && !os.IsNotExist(err) && store.log != nil {
			store.log.Debugw("failed to remove evicted live profile", "file", filename, "error", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("create live profile cache: %w", err)
	}
	store.cache = cache

	store.loadExisting()
	return store, nil
}

func (s *liveProfileStore) Record(query string, profile map[string]interface{}) {
	if s == nil || query == "" {
		return
	}

	key := liveProfileKey(query)
	filename := filepath.Join(s.dir, liveProfileFilename(key))

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		if s.log != nil {
			s.log.Debugw("failed to marshal live query profile", "error", err)
		}
		return
	}

	if err := s.writeFileAtomic(filename, data); err != nil {
		if s.log != nil {
			s.log.Debugw("failed to write live query profile", "file", filename, "error", err)
		}
		return
	}

	s.mx.Lock()
	s.cache.Add(key, filename)
	s.mx.Unlock()
}

func (s *liveProfileStore) RecordLiveProfile(query string, profile map[string]interface{}) {
	s.Record(query, profile)
}

func (s *liveProfileStore) List() []map[string]interface{} {
	if s == nil {
		return nil
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if s.log != nil {
			s.log.Debugw("failed to read live profile directory", "dir", s.dir, "error", err)
		}
		return nil
	}

	type profileEntry struct {
		modTime int64
		data    map[string]interface{}
	}

	var results []profileEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, liveProfileFilePrefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(s.dir, name)
		bytes, err := os.ReadFile(path)
		if err != nil {
			if s.log != nil {
				s.log.Debugw("failed to read live query profile", "file", path, "error", err)
			}
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(bytes, &payload); err != nil {
			if s.log != nil {
				s.log.Debugw("failed to unmarshal live query profile", "file", path, "error", err)
			}
			continue
		}
		info, err := entry.Info()
		if err != nil {
			if s.log != nil {
				s.log.Debugw("failed to stat live query profile", "file", path, "error", err)
			}
			continue
		}
		results = append(results, profileEntry{
			modTime: info.ModTime().UnixNano(),
			data:    payload,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].modTime > results[j].modTime
	})

	profiles := make([]map[string]interface{}, 0, len(results))
	for _, item := range results {
		profiles = append(profiles, item.data)
	}
	return profiles
}

func (s *liveProfileStore) writeFileAtomic(filename string, data []byte) error {
	tmp, err := os.CreateTemp(s.dir, "live-profile-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filename); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

func (s *liveProfileStore) loadExisting() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if s.log != nil {
			s.log.Debugw("failed to scan live profile directory", "dir", s.dir, "error", err)
		}
		return
	}

	type fileEntry struct {
		path    string
		modTime int64
		key     string
	}

	var files []fileEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, liveProfileFilePrefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		key := strings.TrimSuffix(strings.TrimPrefix(name, liveProfileFilePrefix), ".json")
		if key == "" {
			continue
		}
		files = append(files, fileEntry{
			path:    filepath.Join(s.dir, name),
			modTime: info.ModTime().UnixNano(),
			key:     key,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime < files[j].modTime
	})

	s.mx.Lock()
	defer s.mx.Unlock()
	for _, file := range files {
		s.cache.Add(file.key, file.path)
	}
}

func liveProfileKey(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:])
}

func liveProfileFilename(key string) string {
	return fmt.Sprintf("%s%s.json", liveProfileFilePrefix, key)
}
