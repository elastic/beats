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

package ilm

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

type stdSupport struct {
	log *logp.Logger

	mode        Mode
	overwrite   bool
	checkExists bool

	alias  Alias
	policy Policy
}

type stdManager struct {
	*stdSupport
	client ClientHandler

	// cached info
	cache infoCache
}

type infoCache struct {
	LastUpdate time.Time
	Enabled    bool
}

var defaultCacheDuration = 5 * time.Minute

// NewStdSupport creates an instance of default ILM support implementation.
func NewStdSupport(
	log *logp.Logger,
	mode Mode,
	alias Alias,
	policy Policy,
	overwrite, checkExists bool,
) Supporter {
	return &stdSupport{
		log:         log,
		mode:        mode,
		overwrite:   overwrite,
		checkExists: checkExists,
		alias:       alias,
		policy:      policy,
	}
}

func (s *stdSupport) Mode() Mode      { return s.mode }
func (s *stdSupport) Alias() Alias    { return s.alias }
func (s *stdSupport) Policy() Policy  { return s.policy }
func (s *stdSupport) Overwrite() bool { return s.overwrite }

func (s *stdSupport) Manager(h ClientHandler) Manager {
	return &stdManager{
		client:     h,
		stdSupport: s,
	}
}

func (m *stdManager) CheckEnabled() (bool, error) {
	if m.mode == ModeDisabled {
		return false, nil
	}

	if m.cache.Valid() {
		return m.cache.Enabled, nil
	}

	enabled, err := m.client.CheckILMEnabled(m.mode)
	if err != nil {
		return enabled, err
	}

	if !enabled && m.mode == ModeEnabled {
		return false, errOf(ErrESILMDisabled)
	}

	m.cache.Enabled = enabled
	m.cache.LastUpdate = time.Now()
	return enabled, nil
}

func (m *stdManager) EnsureAlias() error {
	if !m.checkExists {
		return nil
	}

	b, err := m.client.HasAlias(m.alias.Name)
	if err != nil {
		return err
	}
	if b {
		return nil
	}

	// This always assume it's a date pattern by sourrounding it by <...>
	return m.client.CreateAlias(m.alias)
}

func (m *stdManager) EnsurePolicy(overwrite bool) (bool, error) {
	log := m.log
	overwrite = overwrite || m.Overwrite()

	exists := true
	if m.checkExists && !overwrite {
		b, err := m.client.HasILMPolicy(m.policy.Name)
		if err != nil {
			return false, err
		}
		exists = b
	}

	if !exists || overwrite {
		return !exists, m.client.CreateILMPolicy(m.policy)
	}

	log.Infof("do not generate ilm policy: exists=%v, overwrite=%v",
		exists, overwrite)
	return false, nil
}

func (c *infoCache) Valid() bool {
	return !c.LastUpdate.IsZero() && time.Since(c.LastUpdate) < defaultCacheDuration
}
