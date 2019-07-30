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
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
)

const initialSize = 20 // initialSize specifies the initial size of the Register.

// Registry is the singleton Register instance where all ModuleFactory's and
// MetricSetFactory's should be registered.
var Registry = NewRegister()

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
	log *logp.Logger
	// Lock to control concurrent read/writes
	lock sync.RWMutex
	// A map of module name to ModuleFactory.
	modules map[string]ModuleFactory
	// A map of module name to nested map of MetricSet name to MetricSetRegistration.
	metricSets map[string]map[string]MetricSetRegistration
	// Additional source of non-registered modules
	secondarySource ModulesSource
}

// ModulesSource contains a source of non-registered modules
type ModulesSource interface {
	Modules() ([]string, error)
	HasModule(module string) bool
	MetricSets(module string) ([]string, error)
	DefaultMetricSets(module string) ([]string, error)
	HasMetricSet(module, name string) bool
	MetricSetRegistration(r *Register, module, name string) (MetricSetRegistration, error)
	String() string
}

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		log:        logp.NewLogger("registry"),
		modules:    make(map[string]ModuleFactory, initialSize),
		metricSets: make(map[string]map[string]MetricSetRegistration, initialSize),
	}
}

// AddModule registers a new ModuleFactory. An error is returned if the
// name is empty, factory is nil, or if a factory has already been registered
// under the name.
func (r *Register) AddModule(name string, factory ModuleFactory) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" {
		return fmt.Errorf("module name is required")
	}

	name = strings.ToLower(name)

	_, exists := r.modules[name]
	if exists {
		return fmt.Errorf("module '%s' is already registered", name)
	}

	if factory == nil {
		return fmt.Errorf("module '%s' cannot be registered with a nil factory", name)
	}

	r.modules[name] = factory
	r.log.Infof("Module registered: %s", name)
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

// addMetricSet registers a new MetricSetFactory. An error is returned if any
// parameter is empty or nil or if a factory has already been registered under
// the name.
func (r *Register) addMetricSet(module, name string, factory MetricSetFactory, options ...MetricSetOption) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if module == "" {
		return fmt.Errorf("module name is required")
	}

	if name == "" {
		return fmt.Errorf("metricset name is required")
	}

	module = strings.ToLower(module)
	name = strings.ToLower(name)

	if metricsets, ok := r.metricSets[module]; !ok {
		r.metricSets[module] = map[string]MetricSetRegistration{}
	} else if _, exists := metricsets[name]; exists {
		return fmt.Errorf("metricset '%s/%s' is already registered", module, name)
	}

	if factory == nil {
		return fmt.Errorf("metricset '%s/%s' cannot be registered with a nil factory", module, name)
	}

	// Set the options.
	msInfo := MetricSetRegistration{Name: name, Factory: factory}
	for _, opt := range options {
		opt(&msInfo)
	}

	r.metricSets[module][name] = msInfo
	r.log.Infof("MetricSet registered: %s/%s", module, name)
	return nil
}

// moduleFactory returns the registered ModuleFactory associated with the
// given name. It returns nil if no ModuleFactory is registered.
func (r *Register) moduleFactory(name string) ModuleFactory {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.modules[strings.ToLower(name)]
}

// metricSetRegistration returns the registration data associated with the given
// metricset name. It returns an error if no metricset is registered.
func (r *Register) metricSetRegistration(module, name string) (MetricSetRegistration, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	module = strings.ToLower(module)
	name = strings.ToLower(name)

	metricSets, exists := r.metricSets[module]
	if exists {
		registration, exists := metricSets[name]
		if exists {
			return registration, nil
		}
	}

	// Fallback to secondary source if module is not registered
	if source := r.secondarySource; source != nil && source.HasMetricSet(module, name) {
		registration, err := source.MetricSetRegistration(r, module, name)
		if err != nil {
			return MetricSetRegistration{}, errors.Wrapf(err, "failed to obtain registration for non-registered metricset '%s/%s'", module, name)
		}
		return registration, nil
	}

	return MetricSetRegistration{}, fmt.Errorf("metricset '%s/%s' not found", module, name)
}

// DefaultMetricSets returns the names of the default MetricSets for a module.
// An error is returned if no default MetricSet is declared or the module does
// not exist.
func (r *Register) DefaultMetricSets(module string) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	module = strings.ToLower(module)

	var defaults []string
	metricSets, exists := r.metricSets[module]
	if exists {
		for _, reg := range metricSets {
			if reg.IsDefault {
				defaults = append(defaults, reg.Name)
			}
		}
	}

	// List also default metrics from secondary sources
	if source := r.secondarySource; source != nil && source.HasModule(module) {
		exists = true
		sourceDefaults, err := source.DefaultMetricSets(module)
		if err != nil {
			r.log.Errorf("Failed to get default metric sets for module '%s' from secondary source: %s", module, err)
		} else if len(sourceDefaults) > 0 {
			defaults = append(defaults, sourceDefaults...)
		}
	}

	if !exists {
		return nil, fmt.Errorf("module '%s' not found", module)
	}
	if len(defaults) == 0 {
		return nil, fmt.Errorf("no default metricset exists for module '%s'", module)
	}
	return defaults, nil
}

// Modules returns the list of module names that are registered
func (r *Register) Modules() []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	var dups = map[string]bool{}

	// For the sake of compatibility, grab modules the old way as well, right from the modules map
	for module := range r.modules {
		dups[module] = true
	}

	// List also modules from secondary sources
	if source := r.secondarySource; source != nil {
		sourceModules, err := source.Modules()
		if err != nil {
			r.log.Errorf("Failed to get modules from secondary source: %s", err)
		} else {
			for _, module := range sourceModules {
				dups[module] = true
			}
		}
	}

	// Grab a more comprehensive list from the metricset keys, then reduce and merge
	for mod := range r.metricSets {
		dups[mod] = true
	}

	modules := make([]string, 0, len(dups))
	for mod := range dups {
		modules = append(modules, mod)
	}

	return modules
}

// MetricSets returns the list of MetricSets registered for a given module
func (r *Register) MetricSets(module string) []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	module = strings.ToLower(module)

	var metricsets []string
	sets, ok := r.metricSets[module]
	if ok {
		metricsets = make([]string, 0, len(sets))
		for name := range sets {
			metricsets = append(metricsets, name)
		}
	}

	// List also metric sets from secondary sources
	if source := r.secondarySource; source != nil && source.HasModule(module) {
		sourceMetricSets, err := source.MetricSets(module)
		if err != nil {
			r.log.Errorf("Failed to get metricsets from secondary source: %s", err)
		}
		metricsets = append(metricsets, sourceMetricSets...)
	}

	return metricsets
}

// SetSecondarySource sets an additional source of modules
func (r *Register) SetSecondarySource(source ModulesSource) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.secondarySource = source
}

// String return a string representation of the registered ModuleFactory's and
// MetricSetFactory's.
func (r *Register) String() string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	var modules []string
	for module := range r.modules {
		modules = append(modules, module)
	}

	var metricSets []string
	for module, m := range r.metricSets {
		for name := range m {
			metricSets = append(metricSets, fmt.Sprintf("%s/%s", module, name))
		}
	}

	var secondarySource string
	if source := r.secondarySource; source != nil {
		secondarySource = ", " + source.String()
	}

	sort.Strings(modules)
	sort.Strings(metricSets)
	return fmt.Sprintf("Register [ModuleFactory:[%s], MetricSetFactory:[%s]%s]",
		strings.Join(modules, ", "), strings.Join(metricSets, ", "), secondarySource)
}
