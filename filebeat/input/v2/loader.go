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

package v2

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/unison"
)

// Loader can be used to create Inputs from configurations.
// The loader is initialized with a list of plugins, and finds the correct plugin
// when a configuration is passed to Configure.
type Loader struct {
	log         *logp.Logger
	registry    map[string]Plugin
	typeField   string
	defaultType string
}

// NewLoader creates a new Loader for configuring inputs from a slice if plugins.
// NewLoader returns a SetupError if invalid plugin configurations or duplicates in the slice are detected.
// The Loader will read the plugin name from the configuration object as is
// configured by typeField. If typeField is empty, it defaults to "type".
func NewLoader(log *logp.Logger, plugins []Plugin, typeField, defaultType string) (*Loader, error) {
	if typeField == "" {
		typeField = "type"
	}

	if errs := validatePlugins(plugins); len(errs) > 0 {
		return nil, &SetupError{errs}
	}

	registry := make(map[string]Plugin, len(plugins))
	for _, p := range plugins {
		registry[p.Name] = p
	}

	return &Loader{
		log:         log,
		registry:    registry,
		typeField:   typeField,
		defaultType: defaultType,
	}, nil
}

// Init runs Init on all InputManagers for all plugins known to the loader.
func (l *Loader) Init(group unison.Group, mode Mode) error {
	for _, p := range l.registry {
		if err := p.Manager.Init(group, mode); err != nil {
			return err
		}
	}
	return nil
}

// Configure creates a new input from a Config object.
// The loader reads the input type name from the cfg object and tries to find a
// matching plugin. If a plugin is found, the plugin it's InputManager is used to create
// the input.
// Returns a LoadError if the input name can not be read from the config or if
// the type does not exist. Error values for Ccnfiguration errors do depend on
// the InputManager.
func (l *Loader) Configure(cfg *conf.C) (Input, error) {
	name, err := cfg.String(l.typeField, -1)
	if err != nil {
		if l.defaultType == "" {
			return nil, &LoadError{
				Reason:  ErrNoInputConfigured,
				Message: fmt.Sprintf("%v setting is missing", l.typeField),
			}
		}
		name = l.defaultType
	}

	p, exists := l.registry[name]
	if !exists {
		return nil, &LoadError{Name: name, Reason: ErrUnknownInput}
	}

	log := l.log.With("input", name, "stability", p.Stability, "deprecated", p.Deprecated)
	switch p.Stability {
	case feature.Experimental:
		log.Warnf("EXPERIMENTAL: The %v input is experimental", name)
	case feature.Beta:
		log.Warnf("BETA: The %v input is beta", name)
	}
	if p.Deprecated {
		log.Warnf("DEPRECATED: The %v input is deprecated", name)
	}

	return p.Manager.Create(cfg)
}

// validatePlugins checks if there are multiple plugins with the same name in
// the registry.
func validatePlugins(plugins []Plugin) []error {
	var errs []error

	counts := map[string]int{}
	for _, p := range plugins {
		counts[p.Name]++
		if err := p.validate(); err != nil {
			errs = append(errs, err)
		}
	}

	for name, count := range counts {
		if count > 1 {
			errs = append(errs, fmt.Errorf("plugin '%v' found %v times", name, count))
		}
	}
	return errs
}
