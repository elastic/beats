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

package metrics

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

// CPU manages the CPU metrics from /proc/stat
// *BSD and and linux only use parts of these,
// but the APIs are similar enough that this is defined here,
// and the code that actually returns metrics to users will be OS-specific
type CPU struct {
	user uint64
	nice uint64
	sys  uint64
	idle uint64
	// Linux and openbsd
	irq uint64
	// Linux only below
	wait    uint64
	softIrq uint64
	stolen  uint64
}

// CPUMetrics carries global and per-core CPU metrics
type CPUMetrics struct {
	totals CPU
	// list carries the same data, broken down by CPU
	// right now, this is entirely used for calculating noramlized CPU values
	// In the future, we can expand this to replace system/core
	list []CPU
}

// Total returns the total CPU time in ticks as scraped by the API
func (cpu CPU) Total() uint64 {
	return cpu.user + cpu.nice + cpu.sys + cpu.idle +
		cpu.wait + cpu.irq + cpu.softIrq + cpu.stolen
}

/*
The below code implements a "metrics tracker" that gives us the ability to
calculate CPU percentages, as we average usage across a time period.
*/

// Monitor is used to monitor the overall CPU usage of the system over time.
type Monitor struct {
	lastSample CPUMetrics
	Hostfs     string
}

// New returns a new CPU metrics monitor
// Hostfs is only relevant on linux and freebsd.
func New(hostfs string) *Monitor {
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

/*
	Ticks(), Percentages(), and NormalizedPercentages()
	are wrappers around OS-specific implementations that are meant to insure
	we only return data that is appropriate for a given OS.

	To implement this API for a new OS, you must supply a Get(string) (CpuMetrics, error) function,
	as well as fillCPUMetrics() and fillTicks()
*/

// NormalizedPercentages fills a given MapStr with normalized CPU usage percentages
func (m *Metrics) NormalizedPercentages(event *common.MapStr) {
	// "normalized" in this sense means when we multiply/subtract by the CPU count, we're getting percentages that amount to the average usage per-cpu, as opposed to system-wide
	normCPU := 1

	timeDelta := m.currentSample.Total() - m.previousSample.Total()
	if timeDelta <= 0 {
		return
	}

	fillCPUMetrics(event, m.currentSample, m.previousSample, normCPU, timeDelta, ".norm.pct")
}

// Percentages fills a given MapStr with CPU usage percentages
func (m *Metrics) Percentages(event *common.MapStr) {
	timeDelta := m.currentSample.Total() - m.previousSample.Total()
	if timeDelta <= 0 {
		return
	}
	// on per-core metrics, is doesn't make sense to have global counts, since the data itself is per-core
	// if this is called from a metric we got with FetchCores(), normalize it.
	normCPU := m.count
	if !m.isTotals {
		normCPU = 1
	}
	fillCPUMetrics(event, m.currentSample, m.previousSample, normCPU, timeDelta, ".pct")
}

// Ticks fills a given MapStr with CPU tick counts
// This value is calculated on windows, and comes directly from OS APIs on other platforms
func (m *Metrics) Ticks(event *common.MapStr) {
	m.currentSample.fillTicks(event)
}

// CPUCount returns the count of CPUs. When available, use this instead of runtime.NumCPU()
func (m *Metrics) CPUCount() int {
	return m.count
}

// cpuMetricTimeDelta is a helper used by fillTicks to calculate the delta between two CPU tick values
func cpuMetricTimeDelta(v0, v1, timeDelta uint64, numCPU int) float64 {
	cpuDelta := int64(v1 - v0)
	pct := float64(cpuDelta) / float64(timeDelta)
	return common.Round(pct*float64(numCPU), common.DefaultDecimalPlacesCount)
}
