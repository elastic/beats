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

package cpu

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// CPU manages the CPU metrics from /proc/stat
// If a given metric isn't available on a given platform,
// The value will be null. All methods that use these fields
// should assume that any value can be null.
// The values are in "ticks", which translates to milliseconds of CPU time
type CPU struct {
	User    opt.Uint `struct:"user,omitempty"`
	Sys     opt.Uint `struct:"system,omitempty"`
	Idle    opt.Uint `struct:"idle,omitempty"`
	Nice    opt.Uint `struct:"nice,omitempty"`    // Linux, Darwin, BSD
	Irq     opt.Uint `struct:"irq,omitempty"`     // Linux and openbsd
	Wait    opt.Uint `struct:"iowait,omitempty"`  // Linux and AIX
	SoftIrq opt.Uint `struct:"softirq,omitempty"` // Linux only
	Stolen  opt.Uint `struct:"steal,omitempty"`   // Linux only
}

// MetricOpts defines the fields that are passed along to the formatted output
type MetricOpts struct {
	Ticks                 bool
	Percentages           bool
	NormalizedPercentages bool
}

// CPUMetrics carries global and per-core CPU metrics
type CPUMetrics struct {
	totals CPU
	// list carries the same data, broken down by CPU
	list []CPU
}

// Total returns the total CPU time in ticks as scraped by the API
func (cpu CPU) Total() uint64 {
	// it's generally safe to blindly sum these up,
	// As we're just trying to get a total of all CPU time.
	return opt.SumOptUint(cpu.User, cpu.Nice, cpu.Sys, cpu.Idle, cpu.Wait, cpu.Irq, cpu.SoftIrq, cpu.Stolen)
}

/*
The below code implements a "metrics tracker" that gives us the ability to
calculate CPU percentages, as we average usage across a time period.
*/

// Monitor is used to monitor the overall CPU usage of the system over time.
type Monitor struct {
	lastSample CPUMetrics
	Hostfs     resolve.Resolver
}

// New returns a new CPU metrics monitor
// Hostfs is only relevant on linux and freebsd.
func New(hostfs resolve.Resolver) *Monitor {
	return &Monitor{Hostfs: hostfs}
}

// Fetch collects a new sample of the CPU usage metrics.
// This will overwrite the currently stored samples.
func (m *Monitor) Fetch() (Metrics, error) {
	metric, err := Get(m.Hostfs)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "Error fetching CPU metrics")
	}

	oldLastSample := m.lastSample
	m.lastSample = metric

	return Metrics{previousSample: oldLastSample.totals, currentSample: metric.totals, count: len(metric.list), isTotals: true}, nil
}

// FetchCores collects a new sample of CPU usage metrics per-core
// This will overwrite the currently stored samples.
func (m *Monitor) FetchCores() ([]Metrics, error) {

	metric, err := Get(m.Hostfs)
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching CPU metrics")
	}

	coreMetrics := make([]Metrics, len(metric.list))
	for i := 0; i < len(metric.list); i++ {
		lastMetric := CPU{}
		// Count of CPUs can change
		if len(m.lastSample.list) > i {
			lastMetric = m.lastSample.list[i]
		}
		coreMetrics[i] = Metrics{
			currentSample:  metric.list[i],
			previousSample: lastMetric,
			isTotals:       false,
		}
	}
	m.lastSample = metric
	return coreMetrics, nil
}

// Metrics stores the current and the last sample collected by a Beat.
type Metrics struct {
	previousSample CPU
	currentSample  CPU
	count          int
	isTotals       bool
}

// Format returns the final MapStr data object for the metrics.
func (metric Metrics) Format(opts MetricOpts) (mapstr.M, error) {

	timeDelta := metric.currentSample.Total() - metric.previousSample.Total()
	if timeDelta <= 0 {
		return nil, errors.New("Previous sample is newer than current sample")
	}
	normCPU := metric.count
	if !metric.isTotals {
		normCPU = 1
	}

	formattedMetrics := mapstr.M{}

	reportOptMetric := func(name string, current, previous opt.Uint, norm int) {
		if !current.IsZero() {
			formattedMetrics[name] = fillMetric(opts, current, previous, timeDelta, norm)
		}
	}

	if opts.Percentages {
		formattedMetrics.Put("total.pct", createTotal(metric.previousSample, metric.currentSample, timeDelta, normCPU))
	}
	if opts.NormalizedPercentages {
		formattedMetrics.Put("total.norm.pct", createTotal(metric.previousSample, metric.currentSample, timeDelta, 1))
	}

	reportOptMetric("user", metric.currentSample.User, metric.previousSample.User, normCPU)
	reportOptMetric("system", metric.currentSample.Sys, metric.previousSample.Sys, normCPU)
	reportOptMetric("idle", metric.currentSample.Idle, metric.previousSample.Idle, normCPU)
	reportOptMetric("nice", metric.currentSample.Nice, metric.previousSample.Nice, normCPU)
	reportOptMetric("irq", metric.currentSample.Irq, metric.previousSample.Irq, normCPU)
	reportOptMetric("iowait", metric.currentSample.Wait, metric.previousSample.Wait, normCPU)
	reportOptMetric("softirq", metric.currentSample.SoftIrq, metric.previousSample.SoftIrq, normCPU)
	reportOptMetric("steal", metric.currentSample.Stolen, metric.previousSample.Stolen, normCPU)

	return formattedMetrics, nil
}

func createTotal(prev, cur CPU, timeDelta uint64, numCPU int) float64 {
	idleTime := cpuMetricTimeDelta(prev.Idle, cur.Idle, timeDelta, numCPU)
	// Subtract wait time from total
	// Wait time is not counted from the total as per #7627.
	if !cur.Wait.IsZero() {
		idleTime = idleTime + cpuMetricTimeDelta(prev.Wait, cur.Wait, timeDelta, numCPU)
	}
	return common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)
}

func fillMetric(opts MetricOpts, cur, prev opt.Uint, timeDelta uint64, numCPU int) mapstr.M {
	event := mapstr.M{}
	if opts.Ticks {
		event.Put("ticks", cur.ValueOr(0))
	}
	if opts.Percentages {
		event.Put("pct", cpuMetricTimeDelta(prev, cur, timeDelta, numCPU))
	}
	if opts.NormalizedPercentages {
		event.Put("norm.pct", cpuMetricTimeDelta(prev, cur, timeDelta, 1))
	}

	return event
}

// CPUCount returns the count of CPUs. When available, use this instead of runtime.NumCPU()
func (m *Metrics) CPUCount() int {
	return m.count
}

// cpuMetricTimeDelta is a helper used by fillTicks to calculate the delta between two CPU tick values
func cpuMetricTimeDelta(prev, current opt.Uint, timeDelta uint64, numCPU int) float64 {
	cpuDelta := int64(current.ValueOr(0) - prev.ValueOr(0))
	pct := float64(cpuDelta) / float64(timeDelta)
	return common.Round(pct*float64(numCPU), common.DefaultDecimalPlacesCount)
}
