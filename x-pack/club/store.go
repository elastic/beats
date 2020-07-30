package main

import (
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
)

type kvStore struct {
	registry      *statestore.Registry
	storeName     string
	cleanInterval time.Duration
}

type kvStoreSettings struct {
	Path          string        `config:"path"`
	Permissions   os.FileMode   `config:"file_permissions"`
	CleanInterval time.Duration `config:"cleanup_interval"`
}

func newKVStore(info beat.Info, logger *logp.Logger, cfg kvStoreSettings) (*kvStore, error) {
	memlog, err := memlog.New(logger, memlog.Settings{
		Root:     paths.Resolve(paths.Data, cfg.Path),
		FileMode: cfg.Permissions,
	})
	if err != nil {
		return nil, err
	}

	return &kvStore{
		registry:      statestore.NewRegistry(memlog),
		storeName:     info.Beat,
		cleanInterval: cfg.CleanInterval,
	}, nil
}

func (s *kvStore) Close() {
	s.registry.Close()
}

func (s *kvStore) Access() (*statestore.Store, error) {
	return s.registry.Get(s.storeName)
}

func (s *kvStore) CleanupInterval() time.Duration {
	return s.cleanInterval
}
