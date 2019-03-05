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
	"github.com/elastic/beats/libbeat/common"
	mem "github.com/elastic/beats/libbeat/metric/system/memory"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/pkg/errors"
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
}

// New is a mb.MetricSetFactory that returns a memory.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{base}, nil
}

// Fetch fetches memory metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	memStat, err := mem.Get()
	if err != nil {
		r.Error(errors.Wrap(err, "memory"))
		return
	}
	mem.AddMemPercentage(memStat)

	swapStat, err := mem.GetSwap()
	if err != nil {
		r.Error(errors.Wrap(err, "swap"))
		return
	}
	mem.AddSwapPercentage(swapStat)

	memory := common.MapStr{
		"total": memStat.Total,
		"used": common.MapStr{
			"bytes": memStat.Used,
			"pct":   memStat.UsedPercent,
		},
		"free": memStat.Free,
		"actual": common.MapStr{
			"free": memStat.ActualFree,
			"used": common.MapStr{
				"pct":   memStat.ActualUsedPercent,
				"bytes": memStat.ActualUsed,
			},
		},
	}

	swap := common.MapStr{
		"total": swapStat.Total,
		"used": common.MapStr{
			"bytes": swapStat.Used,
			"pct":   swapStat.UsedPercent,
		},
		"free": swapStat.Free,
	}
	memory["swap"] = swap

	hugePagesStat, err := mem.GetHugeTLBPages()
	if err != nil {
		r.Error(errors.Wrap(err, "hugepages"))
		return
	}
	if hugePagesStat != nil {
		mem.AddHugeTLBPagesPercentage(hugePagesStat)
		memory["hugepages"] = common.MapStr{
			"total": hugePagesStat.Total,
			"used": common.MapStr{
				"bytes": hugePagesStat.TotalAllocatedSize,
				"pct":   hugePagesStat.UsedPercent,
			},
			"free":         hugePagesStat.Free,
			"reserved":     hugePagesStat.Reserved,
			"surplus":      hugePagesStat.Surplus,
			"default_size": hugePagesStat.DefaultSize,
		}
	}

	r.Event(mb.Event{
		MetricSetFields: memory,
	})
}
