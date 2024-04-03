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

package lifecycle

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// stdSupport is a config wrapper that carries lifecycle info.
type stdSupport struct {
	log              *logp.Logger
	lifecycleEnabled bool
}

// stdManager creates, checks, and updates lifecycle policies.
type stdManager struct {
	*stdSupport
	client ClientHandler
	cache  infoCache
}

// infoCache stores config relating to caching lifecycle config
type infoCache struct {
	LastUpdate time.Time
	Enabled    bool
}

var defaultCacheDuration = 5 * time.Minute

// NewStdSupport creates an instance of default ILM support implementation.
// This contains only the config, and a manager must be created to write and check
// lifecycle policies. I suspect that with enough time/work, you could merge the stdSupport and stdManager objects
func NewStdSupport(
	log *logp.Logger,
	lifecycleEnabled bool,
) Supporter {
	return &stdSupport{
		log:              log,
		lifecycleEnabled: lifecycleEnabled,
	}
}

// Enabled returns true if either ILM or DSL are enabled
func (s *stdSupport) Enabled() bool { return s.lifecycleEnabled }

// Manager returns a standard support manager. unlike the stdSupport object,
// the manager is capable of writing and checking lifecycle policies.
func (s *stdSupport) Manager(h ClientHandler) Manager {
	return &stdManager{
		client:     h,
		stdSupport: s,
	}
}

// CheckEnabled checks to see if lifecycle management is enabled.
func (m *stdManager) CheckEnabled() (bool, error) {
	ilmEnabled, err := m.client.CheckEnabled()
	if err != nil {
		return ilmEnabled, err
	}

	if m.cache.Valid() {
		return m.cache.Enabled, nil
	}

	m.cache.Enabled = ilmEnabled
	m.cache.LastUpdate = time.Now()
	return ilmEnabled, nil
}

// EnsurePolicy creates the upstream lifecycle policy, depending on if it exists, and if overwrite is set.
// returns true if the policy has been created
func (m *stdManager) EnsurePolicy(overwrite bool) (bool, error) {
	log := m.log
	if !m.client.CheckExists() {
		log.Infof("lifecycle policy is not checked as check_exists is disabled")
		return false, nil
	}
	overwrite = overwrite || m.client.Overwrite()
	name := m.client.PolicyName()

	var exists bool
	if !overwrite {
		var err error
		exists, err = m.client.HasPolicy()
		if err != nil {
			return false, fmt.Errorf("error checking if policy %s exists: %w", name, err)
		}
	}

	switch {
	case exists && !overwrite:
		log.Infof("lifecycle policy %v exists already.", name)
		return false, nil

	case !exists || overwrite:
		err := m.client.CreatePolicyFromConfig()
		if err != nil {
			log.Errorf("lifecycle policy %v creation failed: %v", name, err)
			return false, err
		}

		log.Infof("lifecycle policy %v successfully created.", name)
		return true, err

	default:
		log.Infof("lifecycle policy not created: exists=%v, overwrite=%v.", exists, overwrite)
		return false, nil
	}
}

// Valid returns true if the cache is valid
func (c *infoCache) Valid() bool {
	return !c.LastUpdate.IsZero() && time.Since(c.LastUpdate) < defaultCacheDuration
}
