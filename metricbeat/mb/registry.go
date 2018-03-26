package mb

import (
	"fmt"
	"sort"
	"strings"
	"sync"

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
	// Lock to control concurrent read/writes
	lock sync.RWMutex
	// A map of module name to ModuleFactory.
	modules map[string]ModuleFactory
	// A map of module name to nested map of MetricSet name to MetricSetRegistration.
	metricSets map[string]map[string]MetricSetRegistration
}

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
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
	logp.Info("MetricSet registered: %s/%s", module, name)
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
	if !exists {
		return MetricSetRegistration{}, fmt.Errorf("metricset '%s/%s' is not registered, module not found", module, name)
	}

	registration, exists := metricSets[name]
	if !exists {
		return MetricSetRegistration{}, fmt.Errorf("metricset '%s/%s' is not registered, metricset not found", module, name)
	}

	return registration, nil
}

// DefaultMetricSets returns the names of the default MetricSets for a module.
// An error is returned if no default MetricSet is declared or the module does
// not exist.
func (r *Register) DefaultMetricSets(module string) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	module = strings.ToLower(module)

	metricSets, exists := r.metricSets[module]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found", module)
	}

	var defaults []string
	for _, reg := range metricSets {
		if reg.IsDefault {
			defaults = append(defaults, reg.Name)
		}
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

	modules := make([]string, 0, len(r.modules))
	for module := range r.modules {
		modules = append(modules, module)
	}

	return modules
}

// MetricSets returns the list of MetricSets registered for a given module
func (r *Register) MetricSets(module string) []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	var metricsets []string

	sets, ok := r.metricSets[strings.ToLower(module)]
	if ok {
		metricsets = make([]string, 0, len(sets))
		for name := range sets {
			metricsets = append(metricsets, name)
		}
	}

	return metricsets
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
