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

// Package secretstores replaces references to secrets kept in external secret
// stores, such as HashiCorp Vault, in monitor configurations.
//
// Store types are registered with Register; implementations live in x-pack.
// A monitor enables a store type with an entry in its secret_stores list,
// which also holds the store connection. Each store type defines the syntax
// of its references and replaces them itself. Without a secret_stores entry,
// a monitor config is left as is.
//
// References are only replaced in the configuration copy given to the monitor
// plugin. Everywhere else (config hashing, diagnostics, std fields) they stay
// as text. The values that replaced them are removed from the events and
// errors of the monitor, see Scrubber.
package secretstores

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
)

// ConfigKey is the monitor setting holding the secret store connections.
const ConfigKey = "secret_stores"

// Store replaces the secret references of one configured connection.
// Implementations must be safe for concurrent use, and must never include
// secret values or credentials in returned errors or logs.
type Store interface {
	// Validate checks the references of scope in the monitor config cfg
	// without contacting the secret store.
	Validate(cfg *conf.C, scope Scope) error
	// Replace returns a copy of the monitor config cfg with the references
	// of scope replaced by their secret values, and the others left as they
	// are. cfg must not be modified. It must call Track with ctx for every
	// secret value it puts in the copy. Without references of scope, it must
	// return cfg itself: allowed_hosts only applies to connections whose
	// references were replaced.
	Replace(ctx context.Context, cfg *conf.C, scope Scope) (*conf.C, error)
	// Close clears cached secrets and releases resources held by the store
	// when it is evicted from the cache.
	Close() error
}

// Factory creates a Store from one secret_stores entry. It must validate the
// configuration without contacting the secret store.
type Factory func(cfg *conf.C, logger *logp.Logger) (Store, error)

var storeTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]Factory{}

	// sharedStores is process-wide, so monitors with the same connection share
	// logins and cached values across factories and OTel receivers.
	sharedStores = newStoreCache()
)

// Register makes a secret store type available. It returns an error if the
// type is invalid or already registered, or if factory is nil.
func Register(storeType string, factory Factory) error {
	if !storeTypePattern.MatchString(storeType) {
		return fmt.Errorf("invalid secret store type %q", storeType)
	}
	if factory == nil {
		return fmt.Errorf("secret store type %q needs a factory", storeType)
	}

	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	if _, exists := factories[storeType]; exists {
		return fmt.Errorf("secret store type %q is already registered", storeType)
	}
	factories[storeType] = factory
	return nil
}

func lookupFactory(storeType string) (Factory, bool) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[storeType]
	return f, ok
}

type cachedStore struct {
	store Store
	refs  int
}

// storeCache keeps one reference-counted Store per connection identity.
// Entries are evicted and closed when their last reference is released.
type storeCache struct {
	mu     sync.Mutex
	stores map[string]*cachedStore
}

func newStoreCache() *storeCache {
	return &storeCache{stores: map[string]*cachedStore{}}
}

// get returns the Store for the connection of e, creating it on first use.
func (c *storeCache) get(e entry, logger *logp.Logger) (Store, error) {
	id, err := e.identity()
	if err != nil {
		return nil, fmt.Errorf("%s: %q entry: %w", ConfigKey, e.storeType, err)
	}
	return c.getOrCreate(id, e, logger, 0)
}

// acquire returns the Store for id/e, creating it on first use and incrementing
// its reference count. Each acquire must be matched by a release of id.
func (c *storeCache) acquire(id string, e entry, logger *logp.Logger) (Store, error) {
	return c.getOrCreate(id, e, logger, 1)
}

func (c *storeCache) getOrCreate(id string, e entry, logger *logp.Logger, refDelta int) (Store, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cs, ok := c.stores[id]; ok {
		cs.refs += refDelta
		return cs.store, nil
	}
	s, err := newStore(e, logger)
	if err != nil {
		return nil, err
	}
	c.stores[id] = &cachedStore{store: s, refs: refDelta}
	return s, nil
}

// release decrements the reference count for each id and evicts and closes
// stores that have no remaining references.
func (c *storeCache) release(logger *logp.Logger, ids ...string) {
	for _, id := range ids {
		c.mu.Lock()
		cs, ok := c.stores[id]
		if !ok {
			c.mu.Unlock()
			continue
		}
		cs.refs--
		if cs.refs > 0 {
			c.mu.Unlock()
			continue
		}
		delete(c.stores, id)
		c.mu.Unlock()

		if err := cs.store.Close(); err != nil && logger != nil {
			logger.Warnf("Could not close secret store: %v", err)
		}
	}
}

func newStore(e entry, logger *logp.Logger) (Store, error) {
	factory, ok := lookupFactory(e.storeType)
	if !ok {
		return nil, fmt.Errorf("%s: unknown secret store type %q", ConfigKey, e.storeType)
	}
	s, err := factory(e.config, logger)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid %q entry: %w", ConfigKey, e.storeType, err)
	}
	return s, nil
}
