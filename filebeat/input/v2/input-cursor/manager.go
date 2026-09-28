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

package cursor

import (
	"context"
	"errors"
	"sync"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/statemanager"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// globalCache shares cursor stores across the process.
// Managers with the same cache key share one store and one background cleaner.
// The cache closes the store after all managers and running inputs release it.
var globalCache = statemanager.NewCache[*store](func(s *store) { s.Release() })

// InputManager is used to create, manage, and coordinate stateful inputs and
// their persistent state.
// The InputManager ensures that only one input can be active for a unique source.
// If two inputs have overlapping sources, both can still collect data, but
// only one input will collect from the common source.
//
// The InputManager automatically cleans up old entries without an active
// input, and without any pending update operations for the persistent store.
//
// The Type field is used to create the key name in the persistent store. Users
// are allowed to add a custom per input configuration ID using the `id`
// setting, to collect the same source multiple times, but with different
// state. The key name in the persistent store becomes <Type>-[<ID>]-<Source Name>
type InputManager struct {
	Logger *logp.Logger

	// StateStore gives the InputManager access to the persistent key value store.
	StateStore statestore.States

	// Type must contain the name of the input type. It is used to create the key name
	// for all sources the inputs collect from.
	Type string

	// DefaultCleanTimeout configures the key/value garbage collection interval.
	// The InputManager will only collect keys for the configured 'Type'
	DefaultCleanTimeout time.Duration

	// Configure returns an array of Sources, and a configured Input instances
	// that will be used to collect events from each source.
	Configure func(cfg *conf.C, log *logp.Logger) ([]Source, Input, error)

	// mu protects releases and closed from concurrent access.
	// releases holds one globalCache release function for each store this manager uses.
	// Close clears releases.
	mu       sync.Mutex
	releases map[string]func()
	closed   bool
}

// Source describe a source the input can collect data from.
// The `Name` method must return an unique name, that will be used to identify
// the source in the persistent state store.
type Source interface {
	Name() string
}

var (
	errNoSourceConfigured = errors.New("no source has been configured")
	errNoInputRunner      = errors.New("no input runner available")
)

// cacheKey identifies the cursor store for this input type and ID.
// The key includes the type because each cursor store loads state for one type.
func (cim *InputManager) cacheKey(inputID string) string {
	return cim.StateStore.StoreKey(cim.Type, inputID) + "::" + cim.Type
}

// ensureSetup opens or reuses the store for inputID and returns its cache key.
// Call ensureSetup without holding cim.mu.
// It unlocks cim.mu before globalCache.Acquire, so Close can proceed while Acquire waits.
func (cim *InputManager) ensureSetup(inputID string) (string, error) {
	cim.mu.Lock()
	if cim.closed {
		cim.mu.Unlock()
		return "", errors.New("input manager is closed")
	}
	if cim.DefaultCleanTimeout <= 0 {
		cim.DefaultCleanTimeout = 30 * time.Minute
	}
	key := cim.cacheKey(inputID)
	if _, ok := cim.releases[key]; ok {
		cim.mu.Unlock()
		return key, nil
	}
	log := cim.Logger.With("input_type", cim.Type)
	interval := cim.StateStore.CleanupInterval()
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	cim.mu.Unlock()

	_, release, err := globalCache.Acquire(
		key,
		func() (*store, error) {
			return openStore(log, cim.StateStore, cim.Type, inputID, true)
		},
		func(ctx context.Context, s *store) {
			runCleaner(ctx, log, s, interval)
		},
		nil,
		nil,
	)
	if err != nil {
		return "", err
	}

	cim.mu.Lock()
	defer cim.mu.Unlock()
	if cim.closed {
		release()
		return "", errors.New("input manager is closed")
	}
	if _, ok := cim.releases[key]; ok {
		// Another Create already added this key. Release the extra reference.
		release()
		return key, nil
	}
	if cim.releases == nil {
		cim.releases = make(map[string]func())
	}
	cim.releases[key] = release
	return key, nil
}

// Close releases all stores that this manager uses.
// After the last user releases a store, the cache stops its cleaner and closes it.
// Call Close after all inputs for this manager stop.
func (cim *InputManager) Close() {
	cim.mu.Lock()
	cim.closed = true
	releases := cim.releases
	cim.releases = nil
	cim.mu.Unlock()
	for _, release := range releases {
		release()
	}
}

// Create builds a new v2.Input using the provided Configure function.
// The Input will run a go-routine per source that has been configured.
func (cim *InputManager) Create(config *conf.C) (v2.Input, error) {
	settings := struct {
		ID            string        `config:"id"`
		CleanInactive time.Duration `config:"clean_inactive"`
	}{ID: "", CleanInactive: cim.DefaultCleanTimeout}
	if err := config.Unpack(&settings); err != nil {
		return nil, err
	}

	cacheKey, err := cim.ensureSetup(settings.ID)
	if err != nil {
		return nil, err
	}

	sources, inp, err := cim.Configure(config, cim.Logger)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, errNoSourceConfigured
	}
	if inp == nil {
		return nil, errNoInputRunner
	}

	return &managedInput{
		manager:      cim,
		userID:       settings.ID,
		cacheKey:     cacheKey,
		sources:      sources,
		input:        inp,
		cleanTimeout: settings.CleanInactive,
	}, nil
}

// acquireLease keeps the store open while the caller uses it.
// It returns ok=false if the manager is closed or the cache entry is inactive.
// Call the returned release function exactly once.
func (cim *InputManager) acquireLease(cacheKey string) (*store, func(), bool) {
	cim.mu.Lock()
	closed := cim.closed
	cim.mu.Unlock()
	if closed {
		return nil, func() {}, false
	}
	return globalCache.Lease(cacheKey)
}

// lock gives the caller exclusive access to the cursor state for key.
// The caller must hold a lease to keep store open until it releases the resource.
func lock(ctx v2.Context, store *store, key string) (*resource, error) {
	resource := store.Get(key)
	err := lockResource(ctx.Logger, resource, ctx.Cancelation)
	if err != nil {
		resource.Release()
		return nil, err
	}
	return resource, nil
}

func lockResource(log *logp.Logger, resource *resource, canceler v2.Canceler) error {
	if !resource.lock.TryLock() {
		log.Infof("Resource '%v' currently in use, waiting...", resource.key)
		err := resource.lock.LockContext(canceler)
		if err != nil {
			log.Infof("Input for resource '%v' has been stopped while waiting", resource.key)
			return err
		}
	}
	return nil
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}
