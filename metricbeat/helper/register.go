package helper

import (
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Registry is the singleton Register instance where all modules and metricsets
// should register their factories.
var Registry = Register{}

// Register contains the factory functions for creating new Modulers and new
// MetricSeters.
type Register struct {
	Modulers     map[string]func() Moduler                // A map of module name to Moduler factory function.
	MetricSeters map[string]map[string]func() MetricSeter // A map of module name to nested map of metricset name to MetricSeter factory function.
}

// AddModuler registers a new Moduler factory. An error is returned if the
// name is empty, factory is nil, or if a factory has already been registered
// under the name.
func (r *Register) AddModuler(name string, factory func() Moduler) error {
	if r.Modulers == nil {
		r.Modulers = map[string]func() Moduler{}
	}

	if name == "" {
		return fmt.Errorf("module name is required")
	}

	_, exists := r.Modulers[name]
	if exists {
		return fmt.Errorf("module '%s' is already registered", name)
	}

	if factory == nil {
		return fmt.Errorf("module '%s' cannot be registered with a nil factory", name)
	}

	r.Modulers[name] = factory
	logp.Info("Module registered: %s", name)
	return nil
}

// AddMetricSeter registers a new MetricSeter factory. An error is returned if
// any parameter is empty or nil or if a factory has already been registered
// under the name.
func (r *Register) AddMetricSeter(module string, name string, factory func() MetricSeter) error {
	if r.MetricSeters == nil {
		r.MetricSeters = map[string]map[string]func() MetricSeter{}
	}

	if module == "" {
		return fmt.Errorf("module name is required")
	}

	if name == "" {
		return fmt.Errorf("metricset name is required")
	}

	if metricsets, ok := r.MetricSeters[module]; !ok {
		r.MetricSeters[module] = map[string]func() MetricSeter{}
	} else if _, exists := metricsets[name]; exists {
		return fmt.Errorf("metricset '%s/%s' is already registered", module, name)
	}

	if factory == nil {
		return fmt.Errorf("metricset '%s/%s' cannot be registered with a nil factory", module, name)
	}

	r.MetricSeters[module][name] = factory
	logp.Info("metricset registered: %s/%s", module, name)
	return nil
}

// GetModule returns a new Module instance for the given moduler name. An
// error is returned if the module does not exist.
func (r *Register) GetModule(cfg *common.Config) (*Module, error) {
	// Unpack config to get the module name.
	config := struct {
		Module string `config:"module"`
	}{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	moduler, ok := r.Modulers[config.Module]
	if !ok {
		return nil, fmt.Errorf("module '%s' does not exist", config.Module)
	}

	return NewModule(cfg, moduler)
}

// GetMetricSet returns a new MetricSet instance given the module and metricset
// name. An error is returned if the module or metricset do not exist.
func (r *Register) GetMetricSet(module *Module, name string) (*MetricSet, error) {
	metricSets, found := r.MetricSeters[module.name]
	if !found {
		return nil, fmt.Errorf("module '%s' does not exist", module.name)
	}

	factory, found := metricSets[name]
	if !found {
		return nil, fmt.Errorf("metricset '%s/%s' does not exist", module.name, name)
	}

	return NewMetricSet(name, factory, module)
}

// String return a string representation of the registered modules and
// metricsets.
func (r Register) String() string {
	var modules []string
	for module := range r.Modulers {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var metricSets []string
	for module, m := range r.MetricSeters {
		for name := range m {
			metricSets = append(metricSets, fmt.Sprintf("%s/%s", module, name))
		}
	}
	sort.Strings(metricSets)

	return fmt.Sprintf("Register [Modules:[%s], MetricSets:[%s]]",
		strings.Join(modules, ", "), strings.Join(metricSets, ", "))
}
