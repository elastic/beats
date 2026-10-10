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

// Package secretstores resolves references to secrets kept in external secret
// stores, such as HashiCorp Vault, in monitor configurations.
//
// A reference has the form $<type>{<path>#<field>}, for example
// $vault{app/creds#token}. The type selects the store implementation,
// registered with Register. Store implementations live in x-pack. The store
// connection is configured per monitor in the secret_stores list.
//
// References are only replaced in the configuration copy given to the monitor
// plugin. Everywhere else (config hashing, diagnostics, std fields) they stay
// as text: no other configuration layer interprets $<type>{...}.
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

// Store resolves secret references for one configured connection.
// Implementations must be safe for concurrent use, and must never include
// secret values or credentials in returned errors or logs.
type Store interface {
	// Resolve returns the value of field in the secret at path.
	Resolve(ctx context.Context, path, field string) (string, error)
}

// Factory creates a Store from one secret_stores entry. It must validate the
// configuration without contacting the secret store.
type Factory func(cfg *conf.C, logger *logp.Logger) (Store, error)

var storeTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var (
	factoriesMu sync.RWMutex
	factories   = map[string]Factory{}

	// stores keeps one Store per connection identity, so monitors with the
	// same connection share logins and cached values. The number of distinct
	// connections is small, so entries are never evicted.
	storesMu sync.Mutex
	stores   = map[string]Store{}
)

// Register makes a secret store type available. The type is also the prefix
// of its references. It panics if the type is invalid or already registered,
// as it is meant to be called from init functions.
func Register(storeType string, factory Factory) {
	if !storeTypePattern.MatchString(storeType) {
		panic(fmt.Sprintf("invalid secret store type %q", storeType))
	}

	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	if _, exists := factories[storeType]; exists {
		panic(fmt.Sprintf("secret store type %q is already registered", storeType))
	}
	factories[storeType] = factory
}

func lookupFactory(storeType string) (Factory, bool) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[storeType]
	return f, ok
}

// sharedStore returns the Store for the connection identity, creating it on
// first use.
func sharedStore(e entry, logger *logp.Logger) (Store, error) {
	storesMu.Lock()
	defer storesMu.Unlock()

	if s, ok := stores[e.identity]; ok {
		return s, nil
	}
	s, err := newStore(e, logger)
	if err != nil {
		return nil, err
	}
	stores[e.identity] = s
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
