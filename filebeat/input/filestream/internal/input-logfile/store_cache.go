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

package input_logfile

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/elastic/beats/v7/filebeat/input/v2/statemanager"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

// logfileEntry bundles all resources that are shared across InputManager
// instances that map to the same backend key.
type logfileEntry struct {
	store      *store
	ackCH      *updateChan
	ackUpdater *updateWriter
}

// globalCache is the process-level singleton store for filestream inputs.
// All InputManager instances that share a backend key use one entry and one
// background cleaner goroutine. The entry is closed only after the last
// manager releases its reference.
var globalCache = statemanager.NewCache[*logfileEntry](func(e *logfileEntry) {
	e.ackUpdater.Close()
	e.store.Release()
})

// acquireStore returns the shared logfileEntry for a backend, opening it on
// first access. The returned release function must be called exactly once when
// the caller is done with the entry.
func acquireStore(logger *logp.Logger, states statestore.States, prefix string) (*logfileEntry, func(), error) {
	key := states.StoreKey()
	log := logger.
		Named("filestream.store_cache").
		WithLazy(zap.String("filestream_store_key", key))

	interval := states.CleanupInterval()
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	var runFn func(context.Context, *logfileEntry)
	if interval > 0 {
		runFn = func(ctx context.Context, e *logfileEntry) {
			log.Debugw("filestream shared store cleaner started")
			runCleaner(log, ctx, e.store, interval)
			log.Debugw("filestream shared store cleaner stopped")
		}
	}

	entry, release, err := globalCache.Acquire(
		key,
		func() (*logfileEntry, error) {
			log.Debugw("initializing filestream shared store cache entry")
			s, err := openStore(log, states, prefix)
			if err != nil {
				return nil, err
			}
			ackCH := newUpdateChan()
			ackUpdater := newUpdateWriter(s, ackCH)
			return &logfileEntry{store: s, ackCH: ackCH, ackUpdater: ackUpdater}, nil
		},
		runFn,
		func() { log.Debugw("waiting for filestream shared store initialization") },
		func() { log.Debugw("filestream shared store cache hit") },
	)
	if err != nil {
		return nil, nil, err
	}
	return entry, release, nil
}
