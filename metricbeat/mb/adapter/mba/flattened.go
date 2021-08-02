package mba

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
)

type metricsetRunnerFactory struct {
	inner cfgfile.RunnerFactory
}

// MetricsetRunnerFactory creates a RunnerFactory that translates the module name consisting of `<module>.<metricset>` string pair
// into a configuration that contains `metricsets` and `module` as separate settings, similar to the
// configuration expected by metricbeat. The updated configuration is then passed to the inner RunnerFactory.
func MetricsetRunnerFactory(inner cfgfile.RunnerFactory) cfgfile.RunnerFactory {
	return &metricsetRunnerFactory{inner: inner}
}

func (f *metricsetRunnerFactory) Create(p beat.Pipeline, cfg *common.Config) (cfgfile.Runner, error) {
	if err := createMetricsetsConfig(cfg); err != nil {
		return nil, err
	}
	return f.inner.Create(p, cfg)
}

func (f *metricsetRunnerFactory) CheckConfig(cfg *common.Config) error {
	if err := createMetricsetsConfig(cfg); err != nil {
		return err
	}
	return f.inner.CheckConfig(cfg)
}

func createMetricsetsConfig(cfg *common.Config) error {
	tmp := struct {
		Module string `config:"module"     validate:"required"`
	}{}

	if err := cfg.Unpack(&tmp); err != nil {
		return err
	}

	names := strings.SplitN(tmp.Module, ".", 2)
	if len(names) < 2 {
		return fmt.Errorf("module metricset %v unknown", names[0])
	}

	err := cfg.Merge(common.MustNewConfigFrom(map[string]interface{}{
		"module":     names[0],
		"metricsets": []string{names[1]},
	}))
	if err != nil {
		return fmt.Errorf("creating module config: %w", err)
	}

	return nil
}
