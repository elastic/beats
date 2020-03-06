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

package network_summary

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	sysinfo "github.com/elastic/go-sysinfo"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "network_summary", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system network_summary metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	counterInfo, err := fetchNetStats()
	if err != nil {
		return errors.Wrap(err, "Error fetching stats")
	}
	if counterInfo == nil {
		return errors.New("NetworkCounters not available on this platform")
	}

	report.Event(mb.Event{
		MetricSetFields: eventMapping(counterInfo),
	})

	return nil
}

// fetchNetStats wraps the go-sysinfo commands, and returns a NetworkCountersInfo struct if one is available
func fetchNetStats() (*sysinfotypes.NetworkCountersInfo, error) {
	h, err := sysinfo.Host()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read self process information")
	}

	if vmstatHandle, ok := h.(sysinfotypes.NetworkCounters); ok {
		info, err := vmstatHandle.NetworkCounters()
		if err != nil {
			return nil, errors.Wrap(err, "error getting network counters")
		}
		return info, nil
	}

	return nil, nil
}
