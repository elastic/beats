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

//go:build linux
// +build linux

package iostat

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/metric/system/diskio"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("linux", "iostat", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	stats          *diskio.IOStat
	includeDevices []string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The linux iostat metricset is beta.")

	config := struct {
		IncludeDevices []string `config:"iostat.include_devices"`
	}{IncludeDevices: []string{}}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet:  base,
		includeDevices: config.IncludeDevices,
		stats:          diskio.NewDiskIOStat(),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	IOstats, err := diskio.IOCounters(m.includeDevices...)
	if err != nil {
		return errors.Wrap(err, "disk io counters")
	}

	// Sample the current cpu counter
	m.stats.OpenSampling()

	// Store the last cpu counter when finished
	defer m.stats.CloseSampling()

	for _, counters := range IOstats {
		event := common.MapStr{
			"name": counters.Name,
		}
		if counters.SerialNumber != "" {
			event["serial_number"] = counters.SerialNumber
		}
		result, err := m.stats.CalcIOStatistics(counters)
		if err != nil {
			return errors.Wrap(err, "error calculating iostat")
		}
		IOstats := AddLinuxIOStat(result)
		event.DeepUpdate(IOstats)

		isOpen := report.Event(mb.Event{
			MetricSetFields: event,
		})
		if !isOpen {
			return nil
		}
	}
	return nil
}
