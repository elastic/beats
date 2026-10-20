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

package secretstores

import (
	"context"
	"fmt"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Resolver replaces secret references in monitor configurations.
type Resolver struct {
	logger *logp.Logger
}

// NewResolver returns a Resolver.
func NewResolver(logger *logp.Logger) *Resolver {
	if logger == nil {
		logger = logp.NewNopLogger()
	}
	return &Resolver{logger: logger.Named("secretstores")}
}

// Resolve returns a copy of cfg without the secret_stores setting, where each
// configured store, in list order, replaced its references. cfg itself is not
// modified. Without secret_stores, cfg is returned as is.
//
// The Scrubber removes the substituted secret values from what the monitor
// publishes. It is nil if there is nothing to scrub.
func (r *Resolver) Resolve(cfg *conf.C) (*conf.C, *Scrubber, error) {
	entries, err := configEntries(cfg)
	if err != nil || len(entries) == 0 {
		return cfg, nil, err
	}

	resolved, err := withoutSecretStores(cfg)
	if err != nil {
		return nil, nil, err
	}
	var t tracker
	ctx := withTracker(context.Background(), &t)
	for _, e := range entries {
		s, err := sharedStores.get(e, r.logger)
		if err != nil {
			return nil, nil, err
		}
		if resolved, err = s.Replace(ctx, resolved); err != nil {
			return nil, nil, fmt.Errorf("could not resolve %s secret references: %w", e.storeType, err)
		}
	}
	return resolved, t.scrubber(), nil
}

// Validate checks the secret_stores setting and lets each configured store
// check its references, without contacting any secret store.
func (r *Resolver) Validate(cfg *conf.C) error {
	entries, err := configEntries(cfg)
	if err != nil || len(entries) == 0 {
		return err
	}

	monitorCfg, err := withoutSecretStores(cfg)
	if err != nil {
		return err
	}
	for _, e := range entries {
		// A new store, so validation never shares state with running monitors.
		s, err := newStore(e, r.logger)
		if err != nil {
			return err
		}
		if err := s.Validate(monitorCfg); err != nil {
			return fmt.Errorf("invalid %s secret references: %w", e.storeType, err)
		}
	}
	return nil
}

// configEntries returns the secret_stores entries of cfg.
func configEntries(cfg *conf.C) ([]entry, error) {
	if !cfg.HasField(ConfigKey) {
		return nil, nil
	}
	var raw struct {
		Stores any `config:"secret_stores"`
	}
	if err := cfg.Unpack(&raw); err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigKey, err)
	}
	return parseEntries(raw.Stores)
}

// withoutSecretStores returns a copy of cfg without secret_stores, to avoid
// leaking store credentials to the stores and the plugin.
func withoutSecretStores(cfg *conf.C) (*conf.C, error) {
	c := conf.NewConfig()
	if err := c.Merge(cfg); err != nil {
		return nil, fmt.Errorf("could not copy monitor config: %w", err)
	}
	if _, err := c.Remove(ConfigKey, -1); err != nil {
		return nil, fmt.Errorf("could not remove %s: %w", ConfigKey, err)
	}
	return c, nil
}
