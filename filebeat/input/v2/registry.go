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
	"sort"

	"github.com/urso/sderr"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
)

// Registry that store a number of plugins.
// The Registry and Plugin interfaces are used to describe the capabilities of
// an extension point.  Registry and Plugin are not necessarily required to
// create an actual input. It is up to the loader implementation if inputs are
// created via Plugins or not.
type Registry interface {
	Each(func(Plugin) bool)
	Find(name string) (Plugin, bool)
}

// RegistryList combines a list of registries into one registry.
type RegistryList []Registry

// RegistryTree combines multiple registries into a tree.  Registry should not
// be used directly, but can be used as a building block for creating specific
// registries that can accept registries of it's own kind of plugins.
type RegistryTree struct {
	plugins    map[string]Plugin
	registries []Registry
}

// Plugin information. The plugin reports common per plugin details. A plugin
// describes one possible configuration for an extension point. The registry is
// used to provide information about all possible configuration variants of an
// extension point.
type Plugin interface {
	Details() feature.Details
	// TODO: configuration schema
}

var _ Registry = (RegistryList)(nil)
var _ Registry = (*RegistryTree)(nil)

// Add adds another registry to the list.
func (l *RegistryList) Add(reg Registry) {
	*l = append(*l, reg)
}

// Validate checks if the registry is valid and does not contain duplicate
// entries after merging.
func (l RegistryList) Validate() error {
	seen := common.StringSet{}
	dups := map[string]int{}
	l.Each(func(p Plugin) bool {
		name := p.Details().Name
		if seen.Has(name) {
			dups[name]++
		}
		seen.Add(name)
		return true
	})

	if len(dups) == 0 {
		return nil
	}

	var errs []error
	for name, count := range dups {
		errs = append(errs, fmt.Errorf("plugin '%v' found %v time(s)", name, count))
	}
	if len(errs) == 1 {
		return errs[0]
	}

	return sderr.WrapAll(errs, "registry has multiple duplicate plugins")
}

// Names returns a sorted list of known plugin names
func (l RegistryList) Names() []string {
	var names []string
	l.Each(func(p Plugin) bool {
		names = append(names, p.Details().Name)
		return true
	})
	sort.Strings(names)
	return names
}

// Find returns the first Plugin matching the given name.
func (l RegistryList) Find(name string) (plugin Plugin, ok bool) {
	for _, reg := range l {
		if p, ok := reg.Find(name); ok {
			return p, ok
		}
	}
	return nil, false
}

// Each iterates over all known plugins
func (l RegistryList) Each(fn func(Plugin) bool) {
	for _, reg := range l {
		reg.Each(fn)
	}
}

// AddPlugin adds another Plugin to the current node in the tree.
func (r *RegistryTree) AddPlugin(p Plugin) error {
	name := p.Details().Name
	if name == "" {
		return ErrPluginWithoutName
	}

	if _, exists := r.Find(name); exists {
		return fmt.Errorf("conflicts with existing '%v' plugin", name)
	}

	if r.plugins == nil {
		r.plugins = map[string]Plugin{}
	}
	r.plugins[name] = p
	return nil
}

// AddRegistry adds a new child node to the current node in the tree.
// It checks that no plugin in r have duplicate names with any plugins
// current accessible.
func (r *RegistryTree) AddRegistry(child Registry) error {
	// check my plugins don't exist already
	var err error
	child.Each(func(p Plugin) bool {
		name := p.Details().Name
		_, exists := r.Find(name)
		if exists {
			err = fmt.Errorf("conflicts with existing '%v' plugin", name)
		}
		return !exists
	})
	if err != nil {
		return err
	}

	r.registries = append(r.registries)
	return nil
}

// Each iterates over all known plugins accessible using this registry.
// The iteration stops when fn return false.
func (r *RegistryTree) Each(fn func(Plugin) bool) {
	var done bool
	for _, reg := range r.registries {
		if done {
			return
		}

		reg.Each(func(p Plugin) bool {
			done = fn(p)
			return done
		})
	}

	for _, p := range r.plugins {
		if done {
			return
		}
		done = fn(p)
	}
}

// Find returns a Plugin based on it's name.
func (r *RegistryTree) Find(name string) (Plugin, bool) {
	if p, ok := r.plugins[name]; ok {
		return p, true
	}

	for _, reg := range r.registries {
		if p, ok := reg.Find(name); ok {
			return p, ok
		}
	}
	return nil, false
}
