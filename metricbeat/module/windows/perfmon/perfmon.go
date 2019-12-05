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

// +build windows

package perfmon

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// CounterConfig for perfmon counters.
type CounterConfig struct {
	InstanceLabel    string `config:"instance_label"    validate:"required"`
	InstanceName     string `config:"instance_name"`
	MeasurementLabel string `config:"measurement_label" validate:"required"`
	Query            string `config:"query"             validate:"required"`
	Format           string `config:"format"`
}

// Config for the windows perfmon metricset.
type Config struct {
	IgnoreNECounters  bool            `config:"perfmon.ignore_non_existent_counters"`
	GroupMeasurements bool            `config:"perfmon.group_measurements_by_instance"`
	CounterConfig     []CounterConfig `config:"perfmon.counters" validate:"required"`
}

func init() {
	mb.Registry.MustAddMetricSet("windows", "perfmon", New)
}

type MetricSet struct {
	mb.BaseMetricSet
	reader *Reader
	log    *logp.Logger
}

// New create a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The perfmon metricset is beta")

	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	for _, value := range config.CounterConfig {
		form := strings.ToLower(value.Format)
		switch form {
		case "", "float":
			value.Format = "float"
		case "long", "large":
		default:
			return nil, errors.Errorf("initialization failed: format '%s' "+
				"for counter '%s' is invalid (must be float, large or long)",
				value.Format, value.InstanceLabel)
		}

	}
	reader, err := NewReader(config)
	if err != nil {
		return nil, errors.Wrap(err, "initialization of reader failed")
	}
	return &MetricSet{
		BaseMetricSet: base,
		reader:        reader,
		log:           logp.NewLogger("perfmon"),
	}, nil
}

// Fetch fetches events and reports them upstream
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// if the ignore_non_existent_counters flag is set and no valid counter paths are found the Read func will still execute, a check is done before
	if len(m.reader.query.counters) == 0 {
		return errors.New("no counters to read")
	}

	// refresh performance counter list
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	// A flag is set if the second call has been executed else refresh will fail (reader.executed)
	if m.reader.executed {
		err := m.reader.RefreshCounterPaths()
		if err != nil {
			return errors.Wrap(err, "failed retrieving counters")
		}
	}
	events, err := m.reader.Read()
	if err != nil {
		return errors.Wrap(err, "failed reading counters")
	}

	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}
	return nil
}

// Close will be called when metricbeat is stopped, should close the query.
func (m *MetricSet) Close() error {
	err := m.reader.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close pdh query")
	}
	return nil
}
