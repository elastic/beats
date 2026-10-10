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
	"slices"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	ucfg "github.com/elastic/go-ucfg"
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

// Resolve returns a copy of cfg with every secret reference replaced by its
// value, and without the secret_stores setting. cfg itself is not modified.
// When cfg has no references, cfg is returned once secret_stores is valid.
func (r *Resolver) Resolve(cfg *conf.C) (*conf.C, error) {
	refs, err := scanConfig(cfg)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 && !cfg.HasField(ConfigKey) {
		return cfg, nil
	}
	stores, err := r.stores(cfg, refs, sharedStore)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return cfg, nil
	}

	// The process-wide options resolve keystore and environment references.
	var m map[string]any
	if err := cfg.Unpack(&m); err != nil {
		return nil, fmt.Errorf("could not read monitor config: %w", err)
	}
	delete(m, ConfigKey)

	ctx := context.Background()
	_, err = walkStrings(m, func(v string) (string, error) {
		return replaceReferences(v, func(storeType, content string) (string, error) {
			ref, err := ParseReference(storeType, content)
			if err != nil {
				return "", err
			}
			return stores[ref.Type].Resolve(ctx, ref.Path, ref.Field)
		})
	})
	if err != nil {
		return nil, fmt.Errorf("could not resolve secret references: %w", err)
	}

	// No VarExp: resolved values must stay literal. PathSep("") keeps keys
	// such as dotted browser params literal; the map is already nested.
	resolved, err := ucfg.NewFrom(m, ucfg.PathSep(""))
	if err != nil {
		return nil, fmt.Errorf("could not build resolved config: %w", err)
	}
	return (*conf.C)(resolved), nil
}

// Validate checks the secret_stores setting and the secret references in cfg
// without contacting any secret store.
func (r *Resolver) Validate(cfg *conf.C) error {
	refs, err := scanConfig(cfg)
	if err != nil {
		return err
	}
	if len(refs) == 0 && !cfg.HasField(ConfigKey) {
		return nil
	}
	_, err = r.stores(cfg, refs, newStore)
	return err
}

// stores parses the secret_stores entries of cfg and checks that every
// reference has one. getStore builds the stores.
func (r *Resolver) stores(cfg *conf.C, refs []Reference, getStore func(entry, *logp.Logger) (Store, error)) (map[string]Store, error) {
	var raw struct {
		Stores any `config:"secret_stores"`
	}
	if err := cfg.Unpack(&raw); err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigKey, err)
	}
	if err := rejectReferences(raw.Stores); err != nil {
		return nil, err
	}
	entries, err := parseEntries(raw.Stores)
	if err != nil {
		return nil, err
	}

	stores := make(map[string]Store, len(entries))
	for _, e := range entries {
		s, err := getStore(e, r.logger)
		if err != nil {
			return nil, err
		}
		stores[e.storeType] = s
	}

	for _, ref := range refs {
		if _, ok := stores[ref.Type]; !ok {
			return nil, fmt.Errorf("%s: reference %s requires a %q entry in %s", ref.Type, ref, ref.Type, ConfigKey)
		}
	}
	return stores, nil
}

// rejectReferences fails if the secret_stores value contains references: a
// store connection cannot depend on another secret store.
func rejectReferences(stores any) error {
	var found bool
	_, _ = walkStrings(stores, func(s string) (string, error) {
		scanReferences(s, func(string, string) { found = true })
		return s, nil
	})
	if found {
		return fmt.Errorf("%s: secret references are not supported in %s, use the keystore or environment variables", ConfigKey, ConfigKey)
	}
	return nil
}

// scanConfig returns the distinct references in cfg, outside secret_stores,
// without resolving any other reference.
func scanConfig(cfg *conf.C) ([]Reference, error) {
	var m map[string]any
	// ResolveNOOP returns ${...} expressions as text and never fails.
	if err := (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep("."), ucfg.ResolveNOOP); err != nil {
		return nil, fmt.Errorf("could not read monitor config: %w", err)
	}
	delete(m, ConfigKey)

	var refs []Reference
	var parseErr error
	_, _ = walkStrings(m, func(v string) (string, error) {
		scanReferences(v, func(storeType, content string) {
			ref, err := ParseReference(storeType, content)
			if err != nil {
				if parseErr == nil {
					parseErr = err
				}
				return
			}
			if !slices.Contains(refs, ref) {
				refs = append(refs, ref)
			}
		})
		return v, nil
	})
	return refs, parseErr
}
