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
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/filebeat/features"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/es"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

var _ statestore.States = (*filebeatStore)(nil)

type filebeatStore struct {
	registry      *statestore.Registry
	esRegistry    *statestore.Registry
	storeName     string
	cleanInterval time.Duration

	// Notifies the Elasticsearch store about configuration change
	// which is available only after the beat runtime manager connects to the Agent
	// and receives the output configuration
	notifier *es.Notifier
}

func openStateStore(ctx context.Context, info beat.Info, logger *logp.Logger, cfg config.Registry, beatPaths *paths.Path) (*filebeatStore, error) {
	var (
		reg backend.Registry
		err error

		esreg    *es.Registry
		notifier *es.Notifier
	)

	if features.IsElasticsearchStateStoreEnabled() {
		notifier = es.NewNotifier()
		esreg = es.New(ctx, logger, notifier)
	}

	reg, err = memlog.New(logger, memlog.Settings{
		Root:     beatPaths.Resolve(paths.Data, cfg.Path),
		FileMode: cfg.Permissions,
	})
	if err != nil {
		return nil, err
	}

	store := &filebeatStore{
		registry:      statestore.NewRegistry(reg),
		storeName:     info.Beat,
		cleanInterval: cfg.CleanInterval,
		notifier:      notifier,
	}

	if esreg != nil {
		store.esRegistry = statestore.NewRegistry(esreg)
	}

	return store, nil
}

func (s *filebeatStore) Close() {
	s.registry.Close()
}

// StoreFor returns the storage registry depending on the type. Default is the file store.
func (s *filebeatStore) StoreFor(typ string) (*statestore.Store, error) {
	if features.IsElasticsearchStateStoreEnabledForInput(typ) && s.esRegistry != nil {
		return s.esRegistry.Get(s.storeName)
	}
	return s.registry.Get(s.storeName)
}

func (s *filebeatStore) CleanupInterval() time.Duration {
	return s.cleanInterval
}
