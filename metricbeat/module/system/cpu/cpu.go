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

//go:build darwin || freebsd || linux || openbsd || windows || aix

package cpu

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/mapstr"
	metrics "github.com/elastic/elastic-agent-system-metrics/metric/cpu"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "cpu", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system CPU metrics.
type MetricSet struct {
	mb.BaseMetricSet
	opts metrics.MetricOpts
	cpu  *metrics.Monitor
}

// New is a mb.MetricSetFactory that returns a cpu.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	opts, err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating config: %w", err)
	}

	if config.CPUTicks != nil && *config.CPUTicks {
		config.Metrics = append(config.Metrics, "ticks")
	}
	sys := base.Module().(resolve.Resolver)
	return &MetricSet{
		BaseMetricSet: base,
		opts:          opts,
		cpu:           metrics.New(sys),
	}, nil
}

// Fetch fetches CPU metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	sample, err := m.cpu.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch CPU times: %w", err)
	}

	event, err := sample.Format(m.opts)
	if err != nil {
		return fmt.Errorf("error formatting metrics: %w", err)
	}
	event.Put("cores", sample.CPUCount())

	//generate the host fields here, since we don't want users disabling it.
	hostEvent, err := sample.Format(metrics.MetricOpts{NormalizedPercentages: true})
	if err != nil {
		return fmt.Errorf("error creating host fields: %w", err)
	}
	hostFields := mapstr.M{}
	err = copyFieldsOrDefault(hostEvent, hostFields, "total.norm.pct", "host.cpu.usage", 0)
	if err != nil {
		return fmt.Errorf("error fetching normalized CPU percent: %w", err)
	}

	r.Event(mb.Event{
		RootFields:      hostFields,
		MetricSetFields: event,
	})

	return nil
}

// Diagnostics implmements the DiagnosticSet interface
func (m *MetricSet) Diagnostics() []diagnostics.DiagnosticSetup {
	m.Logger().Infof("got DiagnosticSetup request for system/cpu")
	if runtime.GOOS == "linux" {
		return []diagnostics.DiagnosticSetup{
			{
				Name:        "cpu-stat",
				Description: "/proc/stat file",
				Filename:    "stat",
				Callback:    m.fetchRawCPU,
			},
			{
				Name:        "cpu-cpuinfo",
				Description: "/proc/cpuinfo file",
				Filename:    "cpuinfo",
				Callback:    m.fetchCPUInfo,
			},
		}
	}
	return nil

}

func (m *MetricSet) fetchRawCPU() []byte {
	sys := m.BaseMetricSet.Module().(resolve.Resolver)
	return diagnostics.GetRawFileOrErrorString(sys, "/proc/stat")
}

func (m *MetricSet) fetchCPUInfo() []byte {
	sys := m.BaseMetricSet.Module().(resolve.Resolver)
	return diagnostics.GetRawFileOrErrorString(sys, "/proc/cpuinfo")
}

// copyFieldsOrDefault copies the field specified by key to the given map. It will
// overwrite the key if it exists. It will update the map with a default value if
// the key does not exist in the source map.
func copyFieldsOrDefault(from, to mapstr.M, key, newkey string, value interface{}) error {
	v, err := from.GetValue(key)
	if errors.Is(err, mapstr.ErrKeyNotFound) {
		_, err = to.Put(newkey, value)
		return err
	}
	if err != nil {
		return err
	}
	_, err = to.Put(newkey, v)
	return err

}
