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

//go:build darwin || linux || openbsd || windows || (freebsd && cgo)
// +build darwin linux openbsd windows freebsd,cgo

package uptime

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
	sigar "github.com/elastic/gosigar"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "uptime", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching an OS uptime metric.
type MetricSet struct {
	mb.BaseMetricSet
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{base}, nil
}

// Fetch fetches the uptime metric from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	var uptime sigar.Uptime
	if err := uptime.Get(); err != nil {
		return errors.Wrap(err, "failed to get uptime")
	}

	r.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"duration": common.MapStr{
				"ms": int64(uptime.Length * 1000),
			},
		},
	})

	return nil
}
