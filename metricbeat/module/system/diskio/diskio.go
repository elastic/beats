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

// +build darwin,cgo freebsd linux windows

package diskio

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"

	"github.com/pkg/errors"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "diskio", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet for fetching system disk IO metrics.
type MetricSet struct {
	mb.BaseMetricSet
	statistics     *DiskIOStat
	includeDevices []string
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		IncludeDevices []string `config:"diskio.include_devices"`
	}{IncludeDevices: []string{}}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet:  base,
		statistics:     NewDiskIOStat(),
		includeDevices: config.IncludeDevices,
	}, nil
}

// Fetch fetches disk IO metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := IOCounters(m.includeDevices...)
	if err != nil {
		return errors.Wrap(err, "disk io counters")
	}

	// Sample the current cpu counter
	m.statistics.OpenSampling()

	// Store the last cpu counter when finished
	defer m.statistics.CloseSampling()

	for _, counters := range stats {
		event := common.MapStr{
			"name": counters.Name,
			"read": common.MapStr{
				"count": counters.ReadCount,
				"time":  counters.ReadTime,
				"bytes": counters.ReadBytes,
			},
			"write": common.MapStr{
				"count": counters.WriteCount,
				"time":  counters.WriteTime,
				"bytes": counters.WriteBytes,
			},
			"io": common.MapStr{
				"time": counters.IoTime,
			},
		}
		var extraMetrics DiskIOMetric
		err := m.statistics.CalIOStatistics(&extraMetrics, counters)
		if err == nil {
			event["iostat"] = common.MapStr{
				"read": common.MapStr{
					"request": common.MapStr{
						"merges_per_sec": extraMetrics.ReadRequestMergeCountPerSec,
						"per_sec":        extraMetrics.ReadRequestCountPerSec,
					},
					"per_sec": common.MapStr{
						"bytes": extraMetrics.ReadBytesPerSec,
					},
					"await": extraMetrics.AvgReadAwaitTime,
				},
				"write": common.MapStr{
					"request": common.MapStr{
						"merges_per_sec": extraMetrics.WriteRequestMergeCountPerSec,
						"per_sec":        extraMetrics.WriteRequestCountPerSec,
					},
					"per_sec": common.MapStr{
						"bytes": extraMetrics.WriteBytesPerSec,
					},
					"await": extraMetrics.AvgWriteAwaitTime,
				},
				"queue": common.MapStr{
					"avg_size": extraMetrics.AvgQueueSize,
				},
				"request": common.MapStr{
					"avg_size": extraMetrics.AvgRequestSize,
				},
				"await":        extraMetrics.AvgAwaitTime,
				"service_time": extraMetrics.AvgServiceTime,
				"busy":         extraMetrics.BusyPct,
			}
		}

		if counters.SerialNumber != "" {
			event["serial_number"] = counters.SerialNumber
		}

		isOpen := r.Event(mb.Event{
			MetricSetFields: event,
		})
		if !isOpen {
			return nil
		}
	}

	return nil
}
