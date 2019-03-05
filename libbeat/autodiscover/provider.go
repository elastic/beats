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

package autodiscover

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
)

// Provider for autodiscover
type Provider interface {
	cfgfile.Runner
}

// ProviderBuilder creates a new provider based on the given config and returns it
type ProviderBuilder func(bus.Bus, uuid.UUID, *common.Config) (Provider, error)

// AddProvider registers a new ProviderBuilder
func (r *registry) AddProvider(name string, provider ProviderBuilder) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" {
		return fmt.Errorf("provider name is required")
	}

	_, exists := r.providers[name]
	if exists {
		return fmt.Errorf("provider '%s' is already registered", name)
	}

	if provider == nil {
		return fmt.Errorf("provider '%s' cannot be registered with a nil factory", name)
	}

	r.providers[name] = provider
	logp.Debug(debugK, "Provider registered: %s", name)
	return nil
}

// GetProvider returns the provider with the giving name, nil if it doesn't exist
func (r *registry) GetProvider(name string) ProviderBuilder {
	r.lock.RLock()
	defer r.lock.RUnlock()

	name = strings.ToLower(name)
	return r.providers[name]
}

// BuildProvider reads provider configuration and instantiate one
func (r *registry) BuildProvider(bus bus.Bus, c *common.Config) (Provider, error) {
	var config ProviderConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	builder := r.GetProvider(config.Type)
	if builder == nil {
		return nil, fmt.Errorf("Unknown autodiscover provider %s", config.Type)
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return builder(bus, uuid, c)
}
