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

package add_kubernetes_metadata

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
)

// Indexing is the singleton Register instance where all Indexers and Matchers
// are stored
var Indexing = NewRegister()

// Register contains Indexer and Matchers to use on pod indexing and event matching
type Register struct {
	sync.RWMutex
	indexers map[string]IndexerConstructor
	matchers map[string]MatcherConstructor

	defaultIndexerConfigs map[string]common.Config
	defaultMatcherConfigs map[string]common.Config
}

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		indexers: make(map[string]IndexerConstructor, 0),
		matchers: make(map[string]MatcherConstructor, 0),

		defaultIndexerConfigs: make(map[string]common.Config, 0),
		defaultMatcherConfigs: make(map[string]common.Config, 0),
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

// AddIndexer to the register
func (r *Register) AddDefaultIndexerConfig(name string, config common.Config) {
	r.defaultIndexerConfigs[name] = config
}

// AddMatcher to the register
func (r *Register) AddDefaultMatcherConfig(name string, config common.Config) {
	r.defaultMatcherConfigs[name] = config
}

// AddIndexer to the register
func (r *Register) GetIndexer(name string) IndexerConstructor {
	indexer, ok := r.indexers[name]
	if ok {
		return indexer
	} else {
		return nil
	}
}

// AddMatcher to the register
func (r *Register) GetMatcher(name string) MatcherConstructor {
	matcher, ok := r.matchers[name]
	if ok {
		return matcher
	} else {
		return nil
	}
}

func (r *Register) GetDefaultIndexerConfigs() map[string]common.Config {
	return r.defaultIndexerConfigs
}

func (r *Register) GetDefaultMatcherConfigs() map[string]common.Config {
	return r.defaultMatcherConfigs
}
