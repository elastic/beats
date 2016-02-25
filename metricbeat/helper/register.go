package helper

import (
	"fmt"
	"github.com/elastic/beats/libbeat/logp"
)

// Global register for moduler and metricseter
// The Register keeps a global list of moduler and metricseter
// A copy of the moduler or metricset instance can be used to create a module or metricset

// TODO: Global variables should be prevent.
// 	This should be moved into the metricbeat object but can't because the init()
//	functions in each metricset are called before the beater object exists.
var Registry = Register{}

type Register struct {
	Modulers     map[string]Moduler
	MetricSeters map[string]map[string]MetricSeter
}

// AddModule registers the given module with the registry
func (r *Register) AddModuler(name string, m Moduler) {

	if r.Modulers == nil {
		r.Modulers = map[string]Moduler{}
	}

	logp.Info("Register module: %s", name)

	r.Modulers[name] = m
}

func (r *Register) AddMetricSeter(module string, name string, m MetricSeter) {

	if r.MetricSeters == nil {
		r.MetricSeters = map[string]map[string]MetricSeter{}
	}

	if _, ok := r.MetricSeters[module]; !ok {
		r.MetricSeters[module] = map[string]MetricSeter{}
	}

	logp.Info("Register metricset %s for module %s", name, module)

	r.MetricSeters[module][name] = m
}

// GetModule returns a new module instance for the given moduler name
func (r *Register) GetModule(config ModuleConfig) (*Module, error) {
	moduler, ok := Registry.Modulers[config.Module]

	if !ok {
		return nil, fmt.Errorf("Module %s does not exist", config.Module)
	}

	return NewModule(config, moduler), nil
}
