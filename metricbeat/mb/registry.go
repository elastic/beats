package mb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
)

const initialSize = 10 // initialSize specifies the initial size of the Register.

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

// Register contains the factory functions for creating new Modules and new
// MetricSets.
type Register struct {
	// A map of module name to ModuleFactory.
	modules map[string]ModuleFactory
	// A map of module name to nested map of MetricSet name to MetricSetFactory.
	metricSets map[string]map[string]MetricSetFactory
}

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		modules:    make(map[string]ModuleFactory, initialSize),
		metricSets: make(map[string]map[string]MetricSetFactory, initialSize),
	}
}

// AddModule registers a new ModuleFactory. An error is returned if the
// name is empty, factory is nil, or if a factory has already been registered
// under the name.
func (r *Register) AddModule(name string, factory ModuleFactory) error {
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
	logp.Info("Module registered: %s", name)
	return nil
}

// AddMetricSet registers a new MetricSetFactory. An error is returned if
// any parameter is empty or nil or if a factory has already been registered
// under the name.
func (r *Register) AddMetricSet(module string, name string, factory MetricSetFactory) error {
	if module == "" {
		return fmt.Errorf("module name is required")
	}

	if name == "" {
		return fmt.Errorf("metricset name is required")
	}

	module = strings.ToLower(module)
	name = strings.ToLower(name)

	if metricsets, ok := r.metricSets[module]; !ok {
		r.metricSets[module] = map[string]MetricSetFactory{}
	} else if _, exists := metricsets[name]; exists {
		return fmt.Errorf("metricset '%s/%s' is already registered", module, name)
	}

	if factory == nil {
		return fmt.Errorf("metricset '%s/%s' cannot be registered with a nil factory", module, name)
	}

	r.metricSets[module][name] = factory
	logp.Info("MetricSet registered: %s/%s", module, name)
	return nil
}

// moduleFactory returns the registered ModuleFactory associated with the
// given name. It returns nil if no ModuleFactory is registered.
func (r *Register) moduleFactory(name string) ModuleFactory {
	return r.modules[strings.ToLower(name)]
}

// metricSetFactory returns the registered MetricSetFactory associated with the
// given name. It returns an error if no MetricSetFactory is registered.
func (r *Register) metricSetFactory(module, name string) (MetricSetFactory, error) {
	module = strings.ToLower(module)
	name = strings.ToLower(name)

	modules, exists := r.metricSets[module]
	if !exists {
		return nil, fmt.Errorf("metricset '%s/%s' is not registered, module not found", module, name)
	}

	f, exists := modules[name]
	if !exists {
		return nil, fmt.Errorf("metricset '%s/%s' is not registered, metricset not found", module, name)
	}

	return f, nil
}

// String return a string representation of the registered ModuleFactory's and
// MetricSetFactory's.
func (r Register) String() string {
	var modules []string
	for module := range r.modules {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var metricSets []string
	for module, m := range r.metricSets {
		for name := range m {
			metricSets = append(metricSets, fmt.Sprintf("%s/%s", module, name))
		}
	}
	sort.Strings(metricSets)

	return fmt.Sprintf("Register [ModuleFactory:[%s], MetricSetFactory:[%s]]",
		strings.Join(modules, ", "), strings.Join(metricSets, ", "))
}
