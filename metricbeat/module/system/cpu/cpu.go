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

// +build darwin freebsd linux openbsd windows

package cpu

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/cpu"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
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
	config Config
	cpu    *cpu.Monitor
}

// New is a mb.MetricSetFactory that returns a cpu.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.CPUTicks != nil && *config.CPUTicks {
		config.Metrics = append(config.Metrics, "ticks")
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		cpu:           new(cpu.Monitor),
	}, nil
}

// Fetch fetches CPU metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	sample, err := m.cpu.Sample()
	if err != nil {
		return errors.Wrap(err, "failed to fetch CPU times")
	}

	r.Event(collectCPUMetrics(m.config.Metrics, sample))

	return nil
}
