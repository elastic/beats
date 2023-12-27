// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"sync"

	conf "github.com/elastic/elastic-agent-libs/config"
)

// Indexing is the singleton Register instance where all Indexers and Matchers
// are stored
var Indexing = NewRegister()

// Register contains Indexer and Matchers to use on allocation indexing and event matching
type Register struct {
	sync.RWMutex
	indexers map[string]IndexerConstructor
	matchers map[string]MatcherConstructor

	defaultIndexerConfigs map[string]conf.C
	defaultMatcherConfigs map[string]conf.C
}

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		indexers: make(map[string]IndexerConstructor, 0),
		matchers: make(map[string]MatcherConstructor, 0),

		defaultIndexerConfigs: make(map[string]conf.C, 0),
		defaultMatcherConfigs: make(map[string]conf.C, 0),
	}
}

// AddIndexer to the register
func (r *Register) AddIndexer(name string, indexer IndexerConstructor) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	r.indexers[name] = indexer
}

// AddMatcher to the register
func (r *Register) AddMatcher(name string, matcher MatcherConstructor) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	r.matchers[name] = matcher
}

// AddDefaultIndexerConfig add default indexer configuration to the register
func (r *Register) AddDefaultIndexerConfig(name string, config conf.C) {
	r.defaultIndexerConfigs[name] = config
}

// AddDefaultMatcherConfig add a default matcher configuration to the register
func (r *Register) AddDefaultMatcherConfig(name string, config conf.C) {
	r.defaultMatcherConfigs[name] = config
}

// GetIndexer by name
func (r *Register) GetIndexer(name string) IndexerConstructor {
	indexer, ok := r.indexers[name]
	if ok {
		return indexer
	}

	return nil
}

// GetMatcher by name
func (r *Register) GetMatcher(name string) MatcherConstructor {
	matcher, ok := r.matchers[name]
	if ok {
		return matcher
	}

	return nil
}

// GetDefaultIndexerConfigs get default indexer configuration
func (r *Register) GetDefaultIndexerConfigs() map[string]conf.C {
	return r.defaultIndexerConfigs
}

// GetDefaultMatcherConfigs get default matcher configuration
func (r *Register) GetDefaultMatcherConfigs() map[string]conf.C {
	return r.defaultMatcherConfigs
}
