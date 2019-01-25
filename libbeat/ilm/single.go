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
	"fmt"
	"net/url"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type ilmSupport struct {
	cfg        Config
	policyName string
	policy     common.MapStr
}

type singlePolicyManager struct {
	*ilmSupport
	client APIHandler

	// cached info
	cache infoCache
}

type infoCache struct {
	LastUpdate time.Time
	Enabled    bool
}

var defaultCacheDuration = 5 * time.Minute

func (s *ilmSupport) Mode() Mode {
	return s.cfg.Mode
}

func (s *ilmSupport) Template() TemplateSettings {
	alias := s.cfg.RolloverAlias
	return TemplateSettings{
		Alias:      alias,
		Pattern:    fmt.Sprintf("%s-*", alias),
		PolicyName: s.policyName,
	}
}

func (s *ilmSupport) Policy() common.MapStr {
	return s.policy
}

func (s *ilmSupport) Manager(h APIHandler) Manager {
	return &singlePolicyManager{
		client:     h,
		ilmSupport: s,
	}
}

func (m *singlePolicyManager) Enabled() (bool, error) {
	if m.cfg.Mode == ModeDisabled {
		return false, nil
	}

	if m.cache.Valid() {
		return m.cache.Enabled, nil
	}

	enabled, err := m.client.ILMEnabled(m.cfg.Mode)
	if err != nil {
		return enabled, err
	}

	m.cache.Enabled = enabled
	m.cache.LastUpdate = time.Now()
	return enabled, nil
}

func (m *singlePolicyManager) EnsureAlias() error {
	alias := m.cfg.RolloverAlias
	b, err := m.client.HasAlias(alias)
	if err != nil {
		return err
	}
	if b {
		return nil
	}

	// Escaping because of date pattern
	pattern := url.PathEscape(m.cfg.Pattern)
	// This always assume it's a date pattern by sourrounding it by <...>
	firstIndex := fmt.Sprintf("%%3C%s-%s%%3E", alias, pattern)
	return m.client.CreateAlias(alias, firstIndex)
}

func (m *singlePolicyManager) EnsurePolicy(overwrite bool) error {
	overwrite = overwrite || m.cfg.Overwrite

	exists := true
	if m.cfg.CheckExists && !overwrite {
		b, err := m.client.HasILMPolicy(m.policyName)
		if err != nil {
			return err
		}
		exists = b
	}

	if !exists || overwrite {
		return m.client.CreateILMPolicy(m.policyName, m.policy)
	} else {
		logp.Info("do not generate ilm policy: exists=%v, overwrite=%v",
			exists, overwrite)
	}
	return nil
}

func (c *infoCache) Valid() bool {
	return !c.LastUpdate.IsZero() && time.Since(c.LastUpdate) < defaultCacheDuration
}
