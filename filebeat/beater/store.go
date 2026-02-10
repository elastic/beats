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

package beater

import (
	"context"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/filebeat/features"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var (
	_ statestore.States = (*filebeatStore)(nil)

	globalMu     sync.Mutex
	globalStores = map[string]*sharedRegistries{}
)

type sharedRegistries struct {
	refCount   int
	registry   *statestore.Registry
	esRegistry *statestore.Registry
	notifier   *es.Notifier
}

type filebeatStore struct {
	shared        *sharedRegistries
	storeName     string
	cleanInterval time.Duration
	path          string // resolved path, key into globalStores

	// Notifies the Elasticsearch store about configuration change
	// which is available only after the beat runtime manager connects to the Agent
	// and receives the output configuration
	notifier *es.Notifier
}

func openStateStore(ctx context.Context, info beat.Info, logger *logp.Logger, cfg config.Registry, beatPaths *paths.Path) (*filebeatStore, error) {
	resolvedPath := beatPaths.Resolve(paths.Data, cfg.Path)

	globalMu.Lock()
	defer globalMu.Unlock()

	shared, ok := globalStores[resolvedPath]
	if !ok {
		reg, err := memlog.New(logger, memlog.Settings{
			Root:     resolvedPath,
			FileMode: cfg.Permissions,
		})
		if err != nil {
			return nil, err
		}

		shared = &sharedRegistries{
			registry: statestore.NewRegistry(reg),
		}

		if features.IsElasticsearchStateStoreEnabled() {
			// The notifier is a concurrency-safe pub/sub broadcaster shared between
			// the es.Registry (subscriber) and all filebeatStore wrappers (publishers).
			// Multiple Notify() calls are idempotent, so sharing across wrappers is safe.
			shared.notifier = es.NewNotifier()
			shared.esRegistry = statestore.NewRegistry(es.New(ctx, logger, shared.notifier))
		}

		globalStores[resolvedPath] = shared
	}

	shared.refCount++

	return &filebeatStore{
		shared:        shared,
		storeName:     info.Beat,
		cleanInterval: cfg.CleanInterval,
		path:          resolvedPath,
		notifier:      shared.notifier,
	}, nil
}

func (s *filebeatStore) Close() {
	globalMu.Lock()
	defer globalMu.Unlock()

	s.shared.refCount--
	if s.shared.refCount == 0 {
		s.shared.registry.Close()
		if s.shared.esRegistry != nil {
			s.shared.esRegistry.Close()
		}
		delete(globalStores, s.path)
	}
}

// StoreFor returns the storage registry depending on the type. Default is the file store.
func (s *filebeatStore) StoreFor(typ string) (*statestore.Store, error) {
	if features.IsElasticsearchStateStoreEnabledForInput(typ) && s.shared.esRegistry != nil {
		return s.shared.esRegistry.Get(s.storeName)
	}
	return s.shared.registry.Get(s.storeName)
}

func (s *filebeatStore) CleanupInterval() time.Duration {
	return s.cleanInterval
}
