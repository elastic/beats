package mb

import (
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var debugf = logp.MakeDebug("mb")

var (
	// ErrEmptyConfig indicates that modules configuration list is nil or empty.
	ErrEmptyConfig = errors.New("one or more modules must be configured")

	// ErrAllModulesDisabled indicates that all modules are disabled. At least
	// one module must be enabled.
	ErrAllModulesDisabled = errors.New("all modules are disabled")

	// ErrModuleDisabled indicates a disabled module has been tried to instantiate.
	ErrModuleDisabled = errors.New("disabled module")
)

// NewModule builds a new Module and its associated MetricSets based on the
// provided configuration data. config contains config data (the data
// will be unpacked into ModuleConfig structs). r is the Register where the
// ModuleFactory's and MetricSetFactory's will be obtained from. This method
// returns a Module and its configured MetricSets or an error.
func NewModule(config *common.Config, r *Register) (Module, []MetricSet, error) {
	if !config.Enabled() {
		return nil, nil, ErrModuleDisabled
	}

	bm, err := newBaseModuleFromConfig(config)
	if err != nil {
		return nil, nil, err
	}

	module, err := createModule(r, bm)
	if err != nil {
		return nil, nil, err
	}

	metricsets, err := initMetricSets(r, module)
	if err != nil {
		return nil, nil, err
	}

	return module, metricsets, nil
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

	// If timeout is not set, timeout is set to the same value as period
	if baseModule.config.Timeout == 0 {
		baseModule.config.Timeout = baseModule.config.Period
	}

	baseModule.name = strings.ToLower(baseModule.config.Module)

	err = mustNotContainDuplicates(baseModule.config.Hosts)
	if err != nil {
		return baseModule, errors.Wrapf(err, "invalid hosts for module '%s'", baseModule.name)
	}

	return baseModule, nil
}

func createModule(r *Register, bm BaseModule) (Module, error) {
	f := r.moduleFactory(bm.Name())
	if f == nil {
		f = DefaultModuleFactory
	}

	return f(bm)
}

func initMetricSets(r *Register, m Module) ([]MetricSet, error) {
	var (
		errs       multierror.Errors
		metricsets []MetricSet
	)

	bms, err := newBaseMetricSets(r, m)
	if err != nil {
		return nil, err
	}

	for _, bm := range bms {
		f, hostParser, err := r.metricSetFactory(bm.Module().Name(), bm.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		bm.hostData = HostData{URI: bm.host}
		if hostParser != nil {
			bm.hostData, err = hostParser(bm.Module(), bm.host)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "host parsing failed for %v-%v",
					bm.Module().Name(), bm.Name()))
				continue
			}
			bm.host = bm.hostData.Host
		}

		metricSet, err := f(bm)
		if err == nil {
			err = mustHaveModule(metricSet, bm)
			if err == nil {
				err = mustImplementFetcher(metricSet)
			}
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}

		metricsets = append(metricsets, metricSet)
	}

	return metricsets, errs.Err()
}

// newBaseMetricSets creates a new BaseMetricSet for all MetricSets defined
// in the module's config. An error is returned if no MetricSets are specified
// in the module's config and no default MetricSet is defined.
func newBaseMetricSets(r *Register, m Module) ([]BaseMetricSet, error) {
	hosts := []string{""}
	if l := m.Config().Hosts; len(l) > 0 {
		hosts = l
	}

	metricSetNames := m.Config().MetricSets
	if len(metricSetNames) == 0 {
		var err error
		metricSetNames, err = r.defaultMetricSets(m.Name())
		if err != nil {
			return nil, errors.Errorf("no metricsets configured for module '%s'", m.Name())
		}
	}

	var metricsets []BaseMetricSet
	for _, name := range metricSetNames {
		name = strings.ToLower(name)
		for _, host := range hosts {
			metricsets = append(metricsets, BaseMetricSet{
				name:   name,
				module: m,
				host:   host,
			})
		}
	}
	return metricsets, nil
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

	if _, ok := ms.(ReportingMetricSet); ok {
		ifcs = append(ifcs, "ReportingMetricSet")
	}

	if _, ok := ms.(PushMetricSet); ok {
		ifcs = append(ifcs, "PushMetricSet")
	}

	switch len(ifcs) {
	case 0:
		return fmt.Errorf("MetricSet '%s/%s' does not implement an event "+
			"producing interface (EventFetcher, EventsFetcher, "+
			"ReportingMetricSet, or PushMetricSet)",
			ms.Module().Name(), ms.Name())
	case 1:
		return nil
	default:
		return fmt.Errorf("MetricSet '%s/%s' can only implement a single "+
			"event producing interface, but implements %v", ms.Module().Name(),
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
