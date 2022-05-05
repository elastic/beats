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

//go:build (darwin && cgo) || freebsd || linux || windows
// +build darwin,cgo freebsd linux windows

package diskio

import (
	"runtime"

	"github.com/elastic/beats/v7/libbeat/metric/system/diskio"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/mapstr"

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
	statistics     *diskio.IOStat
	includeDevices []string
	prevCounters   diskCounter
}

// diskCounter stores previous disk counter values for calculating gauges in next collection
type diskCounter struct {
	prevDiskReadBytes  uint64
	prevDiskWriteBytes uint64
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
		statistics:     diskio.NewDiskIOStat(),
		includeDevices: config.IncludeDevices,
		prevCounters:   diskCounter{},
	}, nil
}

// Fetch fetches disk IO metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := diskio.IOCounters(m.includeDevices...)
	if err != nil {
		return errors.Wrap(err, "disk io counters")
	}

	// Sample the current cpu counter
	m.statistics.OpenSampling()

	// Store the last cpu counter when finished
	defer m.statistics.CloseSampling()

	var diskReadBytes, diskWriteBytes uint64
	for _, counters := range stats {
		event := mapstr.M{
			"name": counters.Name,
			"read": mapstr.M{
				"count": counters.ReadCount,
				"time":  counters.ReadTime,
				"bytes": counters.ReadBytes,
			},
			"write": mapstr.M{
				"count": counters.WriteCount,
				"time":  counters.WriteTime,
				"bytes": counters.WriteBytes,
			},
		}

		// Add linux-only ops in progress
		if runtime.GOOS == "linux" {
			event.Put("io.ops", counters.IopsInProgress)
		}

		// accumulate values from all interfaces
		diskReadBytes += counters.ReadBytes
		diskWriteBytes += counters.WriteBytes

		if runtime.GOOS != "windows" {
			event.Put("io.time", counters.IoTime)
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

	if m.prevCounters != (diskCounter{}) {
		// convert network metrics from counters to gauges
		r.Event(mb.Event{
			RootFields: mapstr.M{
				"host": mapstr.M{
					"disk": mapstr.M{
						"read.bytes":  diskReadBytes - m.prevCounters.prevDiskReadBytes,
						"write.bytes": diskWriteBytes - m.prevCounters.prevDiskWriteBytes,
					},
				},
			},
		})
	}

	// update prevCounters
	m.prevCounters.prevDiskReadBytes = diskReadBytes
	m.prevCounters.prevDiskWriteBytes = diskWriteBytes

	return nil
}
