package mb

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

var debugf = logp.MakeDebug("mb")

var (
	// ErrEmptyConfig indicates that modules configuration list is nil or empty.
	ErrEmptyConfig = errors.New("one or more modules must be configured")

	// ErrAllModulesDisabled indicates that all modules are disabled. At least
	// one module must be enabled.
	ErrAllModulesDisabled = errors.New("all modules are disabled")
)

// NewModules builds new Modules and their associated MetricSets based on the
// provided configuration data. config is a list module config data (the data
// will be unpacked into ModuleConfig structs). r is the Register where the
// ModuleFactory's and MetricSetFactory's will be obtained from. This method
// returns a mapping of Modules to MetricSets or an error.
func NewModules(config []*common.Config, r *Register) (map[Module][]MetricSet, error) {
	if config == nil || len(config) == 0 {
		return nil, ErrEmptyConfig
	}

	baseModules, err := newBaseModulesFromConfig(config)
	if err != nil {
		return nil, err
	}

	// Create new Modules using the registered ModuleFactory's
	modules, err := createModules(r, baseModules)
	if err != nil {
		return nil, err
	}

	// Create new MetricSets for each Module using the registered MetricSetFactory's
	modToMetricSets, err := initMetricSets(r, modules)
	if err != nil {
		return nil, err
	}

	if len(modToMetricSets) == 0 {
		return nil, ErrAllModulesDisabled
	}

	debugf("mb.NewModules() is returning %s", modToMetricSets)
	return modToMetricSets, nil
}

// newBaseModulesFromConfig creates new BaseModules from a list of configs
// each containing ModuleConfig data.
func newBaseModulesFromConfig(config []*common.Config) ([]BaseModule, error) {
	var errs multierror.Errors
	baseModules := make([]BaseModule, 0, len(config))
	for _, rawConfig := range config {
		bm, err := newBaseModuleFromConfig(rawConfig)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if bm.config.Enabled {
			baseModules = append(baseModules, bm)
		}
	}

	return baseModules, errs.Err()
}

// newBaseModuleFromConfig creates a new BaseModule from config. The returned
// BaseModule's name will always be lower case.
func newBaseModuleFromConfig(rawConfig *common.Config) (BaseModule, error) {
	baseModule := BaseModule{
		config:    DefaultModuleConfig(),
		rawConfig: rawConfig,
	}
	err := rawConfig.Unpack(&baseModule.config)
	if err != nil {
		return baseModule, err
	}

	baseModule.name = strings.ToLower(baseModule.config.Module)

	err = mustNotContainDuplicates(baseModule.config.Hosts)
	if err != nil {
		return baseModule, errors.Wrapf(err, "invalid hosts for module '%s'", baseModule.name)
	}

	return baseModule, nil
}

func createModules(r *Register, baseModules []BaseModule) ([]Module, error) {
	modules := make([]Module, 0, len(baseModules))
	var errs multierror.Errors
	for _, bm := range baseModules {
		f := r.moduleFactory(bm.Name())
		if f == nil {
			f = DefaultModuleFactory
		}

		module, err := f(bm)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		modules = append(modules, module)
	}

	err := errs.Err()
	if err != nil {
		return nil, err
	}
	return modules, nil
}

func initMetricSets(r *Register, modules []Module) (map[Module][]MetricSet, error) {
	active := map[Module][]MetricSet{}
	var errs multierror.Errors
	for _, bms := range newBaseMetricSets(modules) {
		f, err := r.metricSetFactory(bms.Module().Name(), bms.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		metricSet, err := f(bms)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = mustImplementFetcher(metricSet)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = mustHaveModule(metricSet, bms)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		module := metricSet.Module()
		active[module] = append(active[module], metricSet)
	}

	err := errs.Err()
	if err != nil {
		return nil, err
	}
	return active, nil

}

// newBaseMetricSets creates a new BaseMetricSet for all MetricSets defined
// in the modules' config.
func newBaseMetricSets(modules []Module) []BaseMetricSet {
	baseMetricSets := make([]BaseMetricSet, 0, len(modules))
	for _, m := range modules {
		hosts := []string{""}
		if len(m.Config().Hosts) > 0 {
			hosts = m.Config().Hosts
		}

		for _, name := range m.Config().MetricSets {
			for _, host := range hosts {
				baseMetricSets = append(baseMetricSets, BaseMetricSet{
					name:   strings.ToLower(name),
					module: m,
					host:   host,
				})
			}
		}
	}
	return baseMetricSets
}

// mustHaveModule returns an error if the given MetricSet's Module() method
// returns nil. This validation ensures that all MetricSet implementations
// honor the interface contract.
func mustHaveModule(ms MetricSet, base BaseMetricSet) error {
	if ms.Module() == nil {
		return fmt.Errorf("%s module cannot be nil in %T", base.module.Name(), ms)
	}
	return nil
}

// mustImplementFetcher returns an error if the given MetricSet does not
// implement one of the Fetcher interface or if it implements more than one
// of them.
func mustImplementFetcher(ms MetricSet) error {
	var ifcs []string
	if _, ok := ms.(EventFetcher); ok {
		ifcs = append(ifcs, "EventFetcher")
	}

	if _, ok := ms.(EventsFetcher); ok {
		ifcs = append(ifcs, "EventsFetcher")
	}

	switch len(ifcs) {
	case 0:
		return fmt.Errorf("MetricSet '%s/%s' does not implement a Fetcher "+
			"interface", ms.Module().Name(), ms.Name())
	case 1:
		return nil
	default:
		return fmt.Errorf("MetricSet '%s/%s' can only implement a single "+
			"Fetcher interface, but implements %v", ms.Module().Name(),
			ms.Name(), ifcs)
	}
}

// mustNotContainDuplicates returns an error if the given slice contains
// duplicate values.
func mustNotContainDuplicates(s []string) error {
	duplicates := map[string]struct{}{}
	set := make(map[string]struct{}, len(s))
	for _, v := range s {
		_, encountered := set[v]
		if encountered {
			duplicates[v] = struct{}{}
			continue
		}
		set[v] = struct{}{}
	}

	if len(duplicates) > 0 {
		var keys []string
		for dup := range duplicates {
			keys = append(keys, dup)
		}
		return fmt.Errorf("duplicates detected [%s]", strings.Join(keys, ", "))
	}

	return nil
}
