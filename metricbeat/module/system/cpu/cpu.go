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

package cpu

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	metrics "github.com/elastic/beats/v7/metricbeat/internal/metrics/cpu"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
		return nil, errors.Wrap(err, "error validating config")
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
		return errors.Wrap(err, "failed to fetch CPU times")
	}

	event, err := sample.Format(m.opts)
	if err != nil {
		return errors.Wrap(err, "error formatting metrics")
	}
	event.Put("cores", sample.CPUCount())

	//generate the host fields here, since we don't want users disabling it.
	hostEvent, err := sample.Format(metrics.MetricOpts{NormalizedPercentages: true})
	if err != nil {
		return errors.Wrap(err, "error creating host fields")
	}
	hostFields := mapstr.M{}
	err = copyFieldsOrDefault(hostEvent, hostFields, "total.norm.pct", "host.cpu.usage", 0)
	if err != nil {
		return errors.Wrap(err, "error fetching normalized CPU percent")
	}

	r.Event(mb.Event{
		RootFields:      hostFields,
		MetricSetFields: event,
	})

	return nil
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
