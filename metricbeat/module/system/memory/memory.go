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

package memory

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	metrics "github.com/elastic/beats/v7/metricbeat/internal/metrics/memory"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/system"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "memory", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system memory metrics.
type MetricSet struct {
	mb.BaseMetricSet
	IsAgent bool
}

// New is a mb.MetricSetFactory that returns a memory.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, fmt.Errorf("unexpected module type")
	}

	return &MetricSet{BaseMetricSet: base, IsAgent: systemModule.IsAgent}, nil
}

// Fetch fetches memory metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {

	eventRaw, err := metrics.Get("")
	if err != nil {
		return errors.Wrap(err, "error fetching memory metrics")
	}

	memory := common.MapStr{}
	err = typeconv.Convert(&memory, &eventRaw)

	// for backwards compatibility, only report if we're not in fleet mode
	// This is entirely linux-specific data that should live in linux/memory.
	// DEPRECATE: remove this for 8.0
	if !m.IsAgent && runtime.GOOS == "linux" {
		err := fetchLinuxMemStats(memory)
		if err != nil {
			return errors.Wrap(err, "error getting page stats")
		}
		vmstat, err := getVMStat()
		if err != nil {
			return errors.Wrap(err, "Error getting VMStat data")
		}
		// Swap in and swap out numbers
		memory.Put("swap.in.pages", vmstat.Pswpin)
		memory.Put("swap.out.pages", vmstat.Pswpout)
		memory.Put("swap.readahead.pages", vmstat.SwapRa)
		memory.Put("swap.readahead.cached", vmstat.SwapRaHit)
	}

	r.Event(mb.Event{
		MetricSetFields: memory,
	})

	return nil
}
