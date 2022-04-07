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

//go:build darwin || freebsd || linux || openbsd || aix
// +build darwin freebsd linux openbsd aix

package load

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/metric/system/cpu"
	"github.com/elastic/beats/v8/libbeat/metric/system/numcpu"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "load", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system CPU load metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New returns a new load MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches system load metrics.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	load, err := cpu.Load()
	if err != nil {
		return errors.Wrap(err, "failed to get CPU load values")
	}

	avgs := load.Averages()
	normAvgs := load.NormalizedAverages()

	event := common.MapStr{
		"cores": numcpu.NumCPU(),
		"1":     avgs.OneMinute,
		"5":     avgs.FiveMinute,
		"15":    avgs.FifteenMinute,
		"norm": common.MapStr{
			"1":  normAvgs.OneMinute,
			"5":  normAvgs.FiveMinute,
			"15": normAvgs.FifteenMinute,
		},
	}

	r.Event(mb.Event{
		MetricSetFields: event,
	})

	return nil
}
