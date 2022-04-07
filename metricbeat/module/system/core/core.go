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
// +build darwin freebsd linux openbsd windows aix

package core

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/metric/system/resolve"
	metrics "github.com/elastic/beats/v8/metricbeat/internal/metrics/cpu"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "core", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet for fetching system core metrics.
type MetricSet struct {
	mb.BaseMetricSet
	opts  metrics.MetricOpts
	cores *metrics.Monitor
}

// New returns a new core MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	opts, err := config.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "error validating config")
	}

	if config.CPUTicks != nil && *config.CPUTicks {
		config.Metrics = append(config.Metrics, "ticks")
	}
	sys := base.Module().(resolve.Resolver)
	return &MetricSet{
		BaseMetricSet: base,
		opts:          opts,
		cores:         metrics.New(sys),
	}, nil
}

// Fetch fetches CPU core metrics from the OS.
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	samples, err := m.cores.FetchCores()
	if err != nil {
		return errors.Wrap(err, "failed to sample CPU core times")

	}

	for id, sample := range samples {
		event, err := sample.Format(m.opts)
		if err != nil {
			return errors.Wrap(err, "error formatting core data")
		}
		event.Put("id", id)

		isOpen := report.Event(mb.Event{
			MetricSetFields: event,
		})
		if !isOpen {
			return nil
		}
	}

	return nil
}
