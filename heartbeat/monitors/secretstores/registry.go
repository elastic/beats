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
// as text.
package secretstores

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
	// are. cfg must not be modified.
	Replace(ctx context.Context, cfg *conf.C, scope Scope) (*conf.C, error)
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

// storeCache keeps one Store per connection identity. The number of distinct
// connections is small, so entries are never evicted.
type storeCache struct {
	mu     sync.Mutex
	stores map[string]Store
}

func newStoreCache() *storeCache {
	return &storeCache{stores: map[string]Store{}}
}

// get returns the Store for the connection of e, creating it on first use.
func (c *storeCache) get(e entry, logger *logp.Logger) (Store, error) {
	id, err := e.identity()
	if err != nil {
		return nil, fmt.Errorf("%s: %q entry: %w", ConfigKey, e.storeType, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.stores[id]; ok {
		return s, nil
	}
	s, err := newStore(e, logger)
	if err != nil {
		return nil, err
	}
	c.stores[id] = s
	return s, nil
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
