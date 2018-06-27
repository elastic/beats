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

package providers

import (
	"fmt"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/feature"
)

// Provider for autodiscover
type Provider interface {
	cfgfile.Runner
}

// Config settings
type Config struct {
	Type string `config:"type"`
}

// Factory creates a new provider based on the given config and returns it
type Factory func(bus.Bus, *common.Config) (Provider, error)

// Namespace is the registry namespace for autodiscover builders.
var Namespace = "libbeat.autodiscover.provider"

// Feature defines a new provider feature.
func Feature(name string, factory Factory, stability feature.Stability) *feature.Feature {
	return feature.New(Namespace, name, factory, stability)
}

// FindFactory find, assert and return a provider factory.
func FindFactory(name string) (Factory, error) {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil, fmt.Errorf("incompatible type for provider, received: '%T'", factory)
	}

	return factory, nil
}

// Build takes the configuration and will create the appropriate provider.
func Build(bus bus.Bus, c *common.Config) (Provider, error) {
	var config Config
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	builder, err := FindFactory(config.Type)
	if err != nil {
		return nil, err
	}
	return builder(bus, c)
}
