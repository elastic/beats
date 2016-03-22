package helper

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
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
	Modulers     map[string]func() Moduler
	MetricSeters map[string]map[string]func() MetricSeter
}

// AddModule registers the given module with the registry
func (r *Register) AddModuler(name string, m func() Moduler) {

	if r.Modulers == nil {
		r.Modulers = map[string]func() Moduler{}
	}

	logp.Info("Register module: %s", name)

	r.Modulers[name] = m
}

func (r *Register) AddMetricSeter(module string, name string, new func() MetricSeter) {

	if r.MetricSeters == nil {
		r.MetricSeters = map[string]map[string]func() MetricSeter{}
	}

	if _, ok := r.MetricSeters[module]; !ok {
		r.MetricSeters[module] = map[string]func() MetricSeter{}
	}

	logp.Info("Register metricset %s for module %s", name, module)

	r.MetricSeters[module][name] = new
}

// GetModule returns a new module instance for the given moduler name
func (r *Register) GetModule(cfg *common.Config) (*Module, error) {

	// Unpack config to load module name
	config := struct {
		Module string `config:"module"`
	}{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	moduler, ok := Registry.Modulers[config.Module]
	if !ok {
		return nil, fmt.Errorf("Module %s does not exist", config.Module)
	}

	return NewModule(cfg, moduler)
}

// GetMetricSet returns a new metricset instance for the given metricset name combined with the module name
func (r *Register) GetMetricSet(module *Module, metricsetName string) (*MetricSet, error) {

	if _, ok := Registry.MetricSeters[module.name]; !ok {
		return nil, fmt.Errorf("Module %s does not exist", module.name)
	}

	if _, ok := Registry.MetricSeters[module.name][metricsetName]; !ok {
		return nil, fmt.Errorf("Metricset %s in module %s does not exist", metricsetName, module.name)
	}

	newMetricSeter := Registry.MetricSeters[module.name][metricsetName]

	return NewMetricSet(metricsetName, newMetricSeter, module)

}
