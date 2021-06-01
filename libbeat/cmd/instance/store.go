package instance

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
)

type beatStateStore struct {
	registry      *statestore.Registry
	storeName     string
	cleanInterval time.Duration
}

func openStateStore(info beat.Info, logger *logp.Logger, cfg StoreSettings) (*beatStateStore, error) {
	memlog, err := memlog.New(logger, memlog.Settings{
		Root:     paths.Resolve(paths.Data, cfg.Path),
		FileMode: cfg.Permissions,
	})
	if err != nil {
		return nil, err
	}

	return &beatStateStore{
		registry:      statestore.NewRegistry(memlog),
		storeName:     info.Beat,
		cleanInterval: cfg.CleanInterval,
	}, nil
}

func (s *beatStateStore) Close() {
	s.registry.Close()
}

func (s *beatStateStore) Access() (*statestore.Store, error) {
	return s.registry.Get(s.storeName)
}

func (s *beatStateStore) CleanupInterval() time.Duration {
	return s.cleanInterval
}
