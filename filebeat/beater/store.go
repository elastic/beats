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
	"os"
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/eslog"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/go-elasticsearch/v8"
)

type filebeatStore struct {
	registry      *statestore.Registry
	storeName     string
	cleanInterval time.Duration
}

func openStateStore(info beat.Info, logger *logp.Logger, cfg config.Registry) (*filebeatStore, error) {
	var backend backend.Registry
	memlog, err := memlog.New(logger, memlog.Settings{
		Root:     paths.Resolve(paths.Data, cfg.Path),
		FileMode: cfg.Permissions,
	})
	if err != nil {
		return nil, err
	}
	backend = memlog
	if os.Getenv("ELASTIC_STATESTORE_ENABLED") == "true" {
		esClient, err := elasticsearch.NewClient(elasticsearch.Config{
			APIKey: os.Getenv("ELASTIC_STATESTORE_API_KEY"),
			Addresses: []string{
				os.Getenv("ELASTIC_STATESTORE_HOST"),
			},
		})
		if err != nil {
			return nil, err
		}
		eslog, err := eslog.New(logger, eslog.Settings{
			IndexPrefix: "filebeat-state",
			ESClient:    esClient,
		})
		if err != nil {
			return nil, err
		}
		backend = eslog
	}

	return &filebeatStore{
		registry:      statestore.NewRegistry(backend),
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
