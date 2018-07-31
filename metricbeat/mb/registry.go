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

package mb

import (
	"fmt"

	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
)

const moduleNamespace = "metricbeat.module"

// Registry is the singleton Register instance where all ModuleFactory's and
// MetricSetFactory's should be registered.
var Registry = NewRegister(feature.Registry)

// DefaultModuleFactory returns the given BaseModule and never returns an error.
// If a MetricSets are registered without an associated ModuleFactory, then
// the DefaultModuleFactory will be used to instantiate a Module.
var DefaultModuleFactory = func(base BaseModule) (Module, error) {
	return &base, nil
}

// ModuleFactory accepts a BaseModule and returns a Module. If there was an
// error creating the Module then an error will be returned.
type ModuleFactory func(base BaseModule) (Module, error)

// MetricSetFactory accepts a BaseMetricSet and returns a MetricSet. If there
// was an error creating the MetricSet then an error will be returned. The
// returned MetricSet must also implement either EventFetcher or EventsFetcher
// (but not both).
type MetricSetFactory func(base BaseMetricSet) (MetricSet, error)

// HostParser is a function that parses a host value from the configuration
// and returns a HostData object. The module is provided in case additional
// configuration values are required to parse and build the HostData object.
// An error should be returned if the host or configuration is invalid.
type HostParser func(module Module, host string) (HostData, error)

// MetricSetRegistration contains the parameters that were used to register
// a MetricSet.
type MetricSetRegistration struct {
	Name    string
	Factory MetricSetFactory

	// Options
	IsDefault  bool
	HostParser HostParser
	Namespace  string
}

// MetricSetOption sets an option for a MetricSetFactory that is being
// registered.
type MetricSetOption func(info *MetricSetRegistration)

// WithHostParser specifies the HostParser that should be used with the
// MetricSet.
func WithHostParser(p HostParser) MetricSetOption {
	return func(r *MetricSetRegistration) {
		r.HostParser = p
	}
}

// DefaultMetricSet specifies that the MetricSetFactory will be the default
// when no MetricSet names are specified in the configuration.
func DefaultMetricSet() MetricSetOption {
	return func(r *MetricSetRegistration) {
		r.IsDefault = true
	}
}

// WithNamespace specifies the fully qualified namespace under which MetricSet
// data will be added. If no namespace is specified then [module].[metricset]
// will be used.
func WithNamespace(namespace string) MetricSetOption {
	return func(r *MetricSetRegistration) {
		r.Namespace = namespace
	}
}

// Register contains the factory functions for creating new Modules and new
// MetricSets. Registers are thread safe for concurrent usage.
type Register struct {
	registry *feature.FeatureRegistry
}

// NewRegister creates and returns a new Register.
func NewRegister(registry *feature.FeatureRegistry) *Register {
	return &Register{
		registry: registry,
	}
}

// AddModule registers a new ModuleFactory. An error is returned if the
// name is empty, factory is nil, or if a factory has already been registered
// under the name.
func (r *Register) AddModule(name string, factory ModuleFactory) error {
	f := feature.New(moduleNamespace, name, factory, feature.NewDetails(name, "", feature.Undefined))
	err := r.registry.Register(f)
	if err != nil {
		return err
	}

	logp.Info("Module registered: %s", name)
	return nil
}

// AddMetricSet registers a new MetricSetFactory. Optionally it accepts a single
// HostParser function for parsing the 'host' configuration data. An error is
// returned if any parameter is empty or nil or if a factory has already been
// registered under the name.
//
// Use MustAddMetricSet for new code.
func (r *Register) AddMetricSet(module string, name string, factory MetricSetFactory, hostParser ...HostParser) error {
	var opts []MetricSetOption
	if len(hostParser) > 0 {
		opts = append(opts, WithHostParser(hostParser[0]))
	}
	return r.addMetricSet(module, name, factory, opts...)
}

// MustAddMetricSet registers a new MetricSetFactory. It panics if any parameter
// is empty or nil OR if a factory has already been registered under this name.
func (r *Register) MustAddMetricSet(module, name string, factory MetricSetFactory, options ...MetricSetOption) {
	if err := r.addMetricSet(module, name, factory, options...); err != nil {
		panic(err)
	}
}

func (r *Register) namespace(name string) string {
	return moduleNamespace + "." + name
}

// addMetricSet registers a new MetricSetFactory. An error is returned if any
// parameter is empty or nil or if a factory has already been registered under
// the name.
func (r *Register) addMetricSet(module, name string, factory MetricSetFactory, options ...MetricSetOption) error {
	if factory == nil {
		return fmt.Errorf("metricset '%s/%s' cannot be registered with a nil factory", module, name)
	}

	// Set the options.
	msInfo := MetricSetRegistration{Name: name, Factory: factory}
	for _, opt := range options {
		opt(&msInfo)
	}

	f := feature.New(
		r.namespace(module),
		name,
		&msInfo,
		feature.NewDetails(name, "", feature.Undefined),
	)

	err := r.registry.Register(f)
	if err != nil {
		return err
	}

	logp.Info("MetricSet registered: %s/%s", module, name)
	return nil
}

// moduleFactory returns the registered ModuleFactory associated with the
// given name. It returns nil if no ModuleFactory is registered.
func (r *Register) moduleFactory(name string) ModuleFactory {
	f, err := r.registry.Lookup(moduleNamespace, name)
	if err != nil {
		return nil
	}

	factory, ok := f.Factory().(ModuleFactory)
	if !ok {
		return nil
	}

	return factory
}

// metricSetRegistration returns the registration data associated with the given
// metricset name. It returns an error if no metricset is registered.
func (r *Register) metricSetRegistration(module, name string) (MetricSetRegistration, error) {
	ms, err := r.registry.Lookup(r.namespace(module), name)
	if err != nil {
		return MetricSetRegistration{}, fmt.Errorf(
			"metricset '%s/%s' is not registered, metricset not found",
			module,
			name,
		)
	}

	registration, ok := ms.Factory().(*MetricSetRegistration)
	if !ok {
		return MetricSetRegistration{}, fmt.Errorf(
			"incompatible type for metricset '%s/%s', type received %T",
			module,
			name,
			ms.Factory(),
		)
	}

	return *registration, nil
}

// DefaultMetricSets returns the names of the default MetricSets for a module.
// An error is returned if no default MetricSet is declared or the module does
// not exist.
func (r *Register) DefaultMetricSets(module string) (defaults []string, err error) {
	// Retrieve all the active metricset under a specific module.
	// Key:
	// metricbeat.module.happroxy.
	// Retrieve:
	// metricbeat.module.happroxy.stats
	features := r.registry.LookupWithPrefix(moduleNamespace + ".")

	if len(features) == 0 {
		return nil, fmt.Errorf("module '%s' not found", module)
	}

	for _, feature := range features {
		ms, ok := feature.Factory().(*MetricSetRegistration)
		if !ok {
			return nil, fmt.Errorf(
				"incompatible type for metricset '%s/%s', type received %T",
				module,
				feature.Name(),
				feature.Factory(),
			)
		}

		if ms.IsDefault {
			defaults = append(defaults, ms.Name)
		}
	}

	return defaults, nil
}

// Modules returns the list of module names that are registered
func (r *Register) Modules() []string {
	allModules, err := r.registry.LookupAll(moduleNamespace)
	if err != nil {
		return []string{}
	}

	modules := make([]string, 0, len(allModules))
	for _, module := range allModules {
		modules = append(modules, module.Name())
	}

	return modules
}

// MetricSets returns the list of MetricSets registered for a given module
func (r *Register) MetricSets(module string) (modules []string) {
	features := r.registry.LookupWithPrefix(r.namespace(module) + ".")

	if len(features) == 0 {
		return modules
	}

	for _, feature := range features {
		ms, ok := feature.Factory().(*MetricSetRegistration)
		if !ok {
			continue
		}

		modules = append(modules, ms.Name)
	}

	return modules
}

// String return a string representation of the registered ModuleFactory's and
// MetricSetFactory's.
func (r *Register) String() string {
	// TODO
	return "TODO"
}
