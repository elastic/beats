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

	"github.com/elastic/beats/v7/libbeat/logp"
)

type stdSupport struct {
	log *logp.Logger

	enabled     bool
	overwrite   bool
	checkExists bool

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
	enabled bool,
	policy Policy,
	overwrite, checkExists bool,
) Supporter {
	return &stdSupport{
		log:         log,
		enabled:     enabled,
		overwrite:   overwrite,
		checkExists: checkExists,
		policy:      policy,
	}
}

func (s *stdSupport) Enabled() bool   { return s.enabled }
func (s *stdSupport) Policy() Policy  { return s.policy }
func (s *stdSupport) Overwrite() bool { return s.overwrite }

func (s *stdSupport) Manager(h ClientHandler) Manager {
	return &stdManager{
		client:     h,
		stdSupport: s,
	}
}

func (m *stdManager) CheckEnabled() (bool, error) {
	if !m.enabled {
		return false, nil
	}

	if m.cache.Valid() {
		return m.cache.Enabled, nil
	}

	ilmEnabled, err := m.client.CheckILMEnabled(m.enabled)
	if err != nil {
		return ilmEnabled, err
	}

	m.cache.Enabled = ilmEnabled
	m.cache.LastUpdate = time.Now()
	return ilmEnabled, nil
}

func (m *stdManager) EnsurePolicy(overwrite bool) (bool, error) {
	log := m.log
	if !m.checkExists {
		log.Infof("ILM policy is not checked as setup.ilm.check_exists is disabled")
		return false, nil
	}

	overwrite = overwrite || m.Overwrite()
	name := m.policy.Name

	var exists bool
	if !overwrite {
		var err error
		exists, err = m.client.HasILMPolicy(name)
		if err != nil {
			return false, err
		}
	}

	switch {
	case exists && !overwrite:
		log.Infof("ILM policy %v exists already.", name)
		return false, nil

	case !exists || overwrite:
		err := m.client.CreateILMPolicy(m.policy)
		if err != nil {
			log.Errorf("ILM policy %v creation failed: %v", name, err)
			return false, err
		}

		log.Infof("ILM policy %v successfully created.", name)
		return true, err

	default:
		log.Infof("ILM policy not created: exists=%v, overwrite=%v.", exists, overwrite)
		return false, nil
	}
}

func (c *infoCache) Valid() bool {
	return !c.LastUpdate.IsZero() && time.Since(c.LastUpdate) < defaultCacheDuration
}
