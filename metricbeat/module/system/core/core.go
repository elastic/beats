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

package core

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/metric/system/cpu"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "core", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet for fetching system core metrics.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	cores  *cpu.CoresMonitor
}

// New returns a new core MetricSet.
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
		cores:         new(cpu.CoresMonitor),
	}, nil
}

// Fetch fetches CPU core metrics from the OS.
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	samples, err := m.cores.Sample()
	if err != nil {
		return errors.Wrap(err, "failed to sample CPU core times")

	}

	for id, sample := range samples {
		event := common.MapStr{"id": id}

		for _, metric := range m.config.Metrics {
			switch strings.ToLower(metric) {
			case percentages:
				// Use NormalizedPercentages here because per core metrics range on [0, 100%].
				pct := sample.Percentages()
				event.Put("user.pct", pct.User)
				event.Put("system.pct", pct.System)
				event.Put("idle.pct", pct.Idle)
				event.Put("iowait.pct", pct.IOWait)
				event.Put("irq.pct", pct.IRQ)
				event.Put("nice.pct", pct.Nice)
				event.Put("softirq.pct", pct.SoftIRQ)
				event.Put("steal.pct", pct.Steal)
			case ticks:
				ticks := sample.Ticks()
				event.Put("user.ticks", ticks.User)
				event.Put("system.ticks", ticks.System)
				event.Put("idle.ticks", ticks.Idle)
				event.Put("iowait.ticks", ticks.IOWait)
				event.Put("irq.ticks", ticks.IRQ)
				event.Put("nice.ticks", ticks.Nice)
				event.Put("softirq.ticks", ticks.SoftIRQ)
				event.Put("steal.ticks", ticks.Steal)
			}
		}

		isOpen := report.Event(mb.Event{
			MetricSetFields: event,
		})
		if !isOpen {
			return nil
		}
	}

	return nil
}
