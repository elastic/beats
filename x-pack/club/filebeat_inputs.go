package main

import (
	"time"

	inputs "github.com/elastic/beats/v7/filebeat/features/input/default-inputs"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
)

type filebeatStore struct {
	registry      *statestore.Registry
	storeName     string
	cleanInterval time.Duration
}

func makeFilebeatRegistry(info beat.Info, log *logp.Logger, store *filebeatStore) v2.Registry {
	inputLogger := log.Named("input")
	v2Inputs, err := v2.PluginRegistry(inputs.Init(info, inputLogger, store))
	if err != nil {
		panic(err)
	}
	return v2Inputs
}

func openStateStore(info beat.Info, logger *logp.Logger, cfg registrySettings) (*filebeatStore, error) {
	memlog, err := memlog.New(logger, memlog.Settings{
		Root:     paths.Resolve(paths.Data, cfg.Path),
		FileMode: cfg.Permissions,
	})
	if err != nil {
		return nil, err
	}

	return &filebeatStore{
		registry:      statestore.NewRegistry(memlog),
		storeName:     info.Beat,
		cleanInterval: cfg.CleanInterval,
	}, nil
}

func (s *filebeatStore) Close() {
	s.registry.Close()
}

func (s *filebeatStore) Access() (*statestore.Store, error) {
	return s.registry.Get(s.storeName)
}

func (s *filebeatStore) CleanupInterval() time.Duration {
	return s.cleanInterval
}
