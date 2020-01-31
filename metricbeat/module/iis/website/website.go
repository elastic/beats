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

package website

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/pkg/errors"
)

// Config for the windows perfmon metricset.
type Config struct {
	Hosts []string `config:"hosts"`
}

// init registers the partition MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("iis", "website", New)
}

// MetricSet type defines all fields of the partition MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	log    *logp.Logger
	reader *Reader
}

// New creates a new instance of the website MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The website metricset is beta")

	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	reader, err := NewReader(config)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger("website"),
		reader:        reader,
	}, nil
}

// Fetch fetches events and reports them upstream
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// if the ignore_non_existent_counters flag is set and no valid counter paths are found the Read func will still execute, a check is done before
	if len(m.reader.query.Counters) == 0 {
		return errors.New("no counters to read")
	}
	var config Config
	if err := m.Module().UnpackConfig(&config); err != nil {
		return nil
	}

	// refresh performance counter list
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	// A flag is set if the second call has been executed else refresh will fail (reader.executed)
	if m.reader.hasRun {
		err := m.reader.InitCounters(config.Hosts)
		if err != nil {
			return errors.Wrap(err, "failed retrieving counters")
		}
	}
	events, err := m.reader.Fetch()
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
