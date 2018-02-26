// +build windows

package perfmon

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
)

// Config for perfmon counters.
type CounterConfig struct {
	InstanceLabel    string `config:"instance_label" validate:"required"`
	InstanceName     string `config:"instance_name"`
	MeasurementLabel string `config:"measurement_label" validate:"required"`
	Query            string `config:"query" validate:"required"`
	Format           string `config:"format"`
}

// Config for the windows perfmon metricset.
type PerfmonConfig struct {
	IgnoreNECounters bool            `config:"perfmon.ignore_non_existent_counters"`
	CounterConfig    []CounterConfig `config:"perfmon.counters" validate:"required"`
}

func init() {
	if err := mb.Registry.AddMetricSet("windows", "perfmon", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	reader *PerfmonReader
}

// New create a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The perfmon metricset is beta")

	config := PerfmonConfig{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	for _, value := range config.CounterConfig {
		form := strings.ToLower(value.Format)
		switch form {
		case "":
			value.Format = "float"
		case "float", "long":
		default:
			err := fmt.Errorf("format '%s' for counter '%s' are not valid", value.Format, value.InstanceLabel)
			return nil, errors.Wrap(err, "initialization failed")
		}

	}

	reader, err := NewPerfmonReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "initialization failed")
	}

	return &MetricSet{
		BaseMetricSet: base,
		reader:        reader,
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	data, err := m.reader.Read()
	if err != nil {
		return nil, errors.Wrap(err, "failed reading counters")
	}

	return data, nil
}
