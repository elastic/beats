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
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
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

func openStateStore(info beat.Info, logger *logp.Logger, cfg config.Registry) (*filebeatStore, error) {
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
