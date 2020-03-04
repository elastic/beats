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

// Package exclinput provides a Loader, Registry, and Plugin definition for
// inputs that require coordinated access to the Filebeat registry using the
// statestore.
package exclinput

import (
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/statestore"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
)

// Loader creates inputs from a configuration by finding and calling the right
// plugin.
type Loader struct {
	typeField    string
	storeConn    *statestore.Connector
	defaultStore string
	reg          *Registry
}

// Registry is used to lookup available plugins.
type Registry v2.RegistryTree

// Plugin to be added to a registry. The Plugin will be used to create an
// Input.
type Plugin struct {
	Name       string
	Stability  feature.Stability
	Deprecated bool
	Info       string
	Doc        string
	Create     func(*common.Config) (Input, error)
}

// Input is created by a Plugin. The Run function is mandatory, while Test is
// optional.
type Input struct {
	Name string
	Run  func(v2.Context, *statestore.Store, beat.PipelineConnector) error
	Test func(v2.TestContext) error
}

// Extension types can be Registry or Plugin. It is used to combine plugins and
// registry into an even bigger registry.
// for Example:
//    r1, _ := NewRegistry(...)
//    r2, _ := NewRegistry(...)
//    r, err := NewRegistry(r1, r2,
//        &Plugin{...},
//        &Plugin{...},
//    )
//    // -> r can be used to load plugins from all registries and plugins
//    //    added.
type Extension interface {
	addToRegistry(reg *Registry) error
}

var _ v2.Loader = (*Loader)(nil)
var _ v2.Plugin = (*Plugin)(nil)
var _ v2.Registry = (*Registry)(nil)
var _ Extension = (*Plugin)(nil)
var _ Extension = (*Registry)(nil)

// NewLoader creates a new Loader for a registry.
func NewLoader(
	storeConn *statestore.Connector,
	defaultStore string,
	reg *Registry,
	typeField string,
) *Loader {
	if typeField == "" {
		typeField = "type"
	}
	return &Loader{
		typeField:    typeField,
		storeConn:    storeConn,
		defaultStore: defaultStore,
		reg:          reg,
	}
}

// Configure looks for a plugin matching the 'type' name in the configuration
// and creates a new Input.  Configure fails if the type is not known, or if
// the plugin can not apply the configuration.
func (l *Loader) Configure(cfg *common.Config) (v2.Input, error) {
	name, err := cfg.String(l.typeField, -1)
	if err != nil {
		return v2.Input{}, err
	}

	plugin, ok := l.reg.findPlugin(name)
	if !ok {
		return v2.Input{}, &v2.LoaderError{Name: name, Reason: v2.ErrUnknown}
	}

	input, err := plugin.Create(cfg)
	if err != nil {
		return v2.Input{}, err
	}

	return v2.Input{
		Name: input.Name,
		Test: input.Test,
		Run: func(ctx v2.Context, conn beat.PipelineConnector) error {
			store, err := l.storeConn.Open(l.defaultStore)
			if err != nil {
				return fmt.Errorf("input could not access the registry store '%v': %+v",
					name, err)
			}
			return input.Run(ctx, store, conn)
		},
	}, nil
}

// NewRegistry creates a new Registry from the list of Registrations and Plugins.
func NewRegistry(exts ...Extension) (*Registry, error) {
	r := &Registry{}
	for _, ext := range exts {
		if err := r.Add(ext); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Add adds another registry or plugin to the current registry.
func (r *Registry) Add(ext Extension) error {
	return ext.addToRegistry(r)
}

func (r *Registry) addToRegistry(parent *Registry) error {
	return (*v2.RegistryTree)(parent).AddRegistry(r)
}

func (p *Plugin) addToRegistry(parent *Registry) error {
	return (*v2.RegistryTree)(parent).AddPlugin(p)
}

// Each iterates over all known plugins accessible using this registry.
// The iteration stops when fn return false.
func (r *Registry) Each(fn func(v2.Plugin) bool) {
	(*v2.RegistryTree)(r).Each(fn)
}

// Find returns a Plugin based on it's name.
func (r *Registry) Find(name string) (plugin v2.Plugin, ok bool) {
	return (*v2.RegistryTree)(r).Find(name)
}

func (r *Registry) findPlugin(name string) (*Plugin, bool) {
	p, ok := r.Find(name)
	if !ok {
		return nil, false
	}
	return p.(*Plugin), ok
}

// Details returns common feature information about the plugin the and input
// type it can generate.
func (p *Plugin) Details() feature.Details {
	return feature.Details{
		Name:       p.Name,
		Stability:  p.Stability,
		Deprecated: p.Deprecated,
		Info:       p.Info,
		Doc:        p.Doc,
	}
}
