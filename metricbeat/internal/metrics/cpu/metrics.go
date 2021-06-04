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
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

// CPU manages the CPU metrics from /proc/stat
// If a given metric isn't available on a given platform,
// The value will be null. All methods that use these fields
// should assume that any value can be null.
// The values are in "ticks", which translates to milliseconds of CPU time
type CPU struct {
	user    *uint64 `metric:"user"`
	sys     *uint64 `metric:"system"`
	idle    *uint64 `metric:"idle"`
	nice    *uint64 `metric:"nice"`    // Linux, Darwin, BSD
	irq     *uint64 `metric:"irq"`     // Linux and openbsd
	wait    *uint64 `metric:"iowait"`  // Linux and AIX
	softIrq *uint64 `metric:"softirq"` // Linux only
	stolen  *uint64 `metric:"steal"`   // Linux only
}

// CPUMetrics carries global and per-core CPU metrics
type CPUMetrics struct {
	totals CPU
	// list carries the same data, broken down by CPU
	list []CPU
}

// Total returns the total CPU time in ticks as scraped by the API
func (cpu CPU) Total() uint64 {

	var total uint64
	fn := func(field uint64, _ int, _ string) {
		total = total + field
	}
	cpu.iterateCPUWithFunc(fn)
	return total

}

// iterateCPUWithFunc uses reflection to interate over the CPU struct, stopping at every non-null field
// `field` is the value of the struct field, iter is the field's place in the struct, and name is the `metric` tag.
func (cpu CPU) iterateCPUWithFunc(iterFunc func(field uint64, iter int, name string)) {
	valueOfCPU := reflect.ValueOf(cpu)
	typeOfCPU := valueOfCPU.Type()
	for i := 0; i < valueOfCPU.NumField(); i++ {
		field := valueOfCPU.Field(i)
		if field.IsNil() {
			continue
		}
		itemValue := field.Elem().Uint()

		var name string
		if tag := typeOfCPU.Field(i).Tag.Get("metric"); tag == "" {
			name = typeOfCPU.Field(i).Name
		} else {
			name = tag
		}
		iterFunc(itemValue, i, name)
	}
}

// fillTicks fills in the map with the raw values from the CPU struct
func (cpu CPU) fillTicks(event *common.MapStr) {

	fn := func(field uint64, _ int, name string) {
		mapName := fmt.Sprintf("%s.ticks", name)
		event.Put(mapName, field)
	}

	cpu.iterateCPUWithFunc(fn)
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

// fillCPUMetrics fills in the given event struct with CPU data from the events.
// Because this code just checks to see what values are null, it's platform independent
func (metrics Metrics) fillCPUMetrics(event *common.MapStr, numCPU int, timeDelta uint64, pathPostfix string) {
	idleTime := cpuMetricTimeDelta(metrics.previousSample.idle, metrics.currentSample.idle, timeDelta, numCPU)

	// Subtract wait time from total
	// Wait time is not counted from the total as per #7627.
	if metrics.currentSample.wait != nil {
		idleTime = idleTime + cpuMetricTimeDelta(metrics.previousSample.wait, metrics.currentSample.wait, timeDelta, numCPU)
	}

	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total"+pathPostfix, totalPct)

	fn := func(field uint64, iter int, name string) {
		mapName := fmt.Sprintf("%s%s", name, pathPostfix)
		valueOfPrevCPU := reflect.ValueOf(metrics.previousSample)
		prevValue := valueOfPrevCPU.Field(iter)
		var prevUint uint64
		if !prevValue.IsNil() {
			prevUint = prevValue.Elem().Uint()
		}
		event.Put(mapName, cpuMetricTimeDelta(&prevUint, &field, timeDelta, numCPU))
	}

	metrics.currentSample.iterateCPUWithFunc(fn)
}

/*
	Ticks(), Percentages(), and NormalizedPercentages()
	are wrappers around OS-specific implementations that are meant to insure
	we only return data that is appropriate for a given OS.

	To implement this API for a new OS, you must supply a Get(string) (CpuMetrics, error) function
	This function should return a CPUMetrics function where any unavailable metric is nil
*/

// NormalizedPercentages fills a given MapStr with normalized CPU usage percentages
func (m *Metrics) NormalizedPercentages(event *common.MapStr) {
	// "normalized" in this sense means when we multiply/subtract by the CPU count, we're getting percentages that amount to the average usage per-cpu, as opposed to system-wide
	normCPU := 1

	timeDelta := m.currentSample.Total() - m.previousSample.Total()
	if timeDelta <= 0 {
		return
	}

	m.fillCPUMetrics(event, normCPU, timeDelta, ".norm.pct")
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
	m.fillCPUMetrics(event, normCPU, timeDelta, ".pct")
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
func cpuMetricTimeDelta(v0, v1 *uint64, timeDelta uint64, numCPU int) float64 {
	var prev, current uint64
	if v0 == nil {
		prev = 0
	} else {
		prev = *v0
	}
	if v1 == nil {
		current = 0
	} else {
		current = *v1
	}
	cpuDelta := int64(current - prev)
	pct := float64(cpuDelta) / float64(timeDelta)
	return common.Round(pct*float64(numCPU), common.DefaultDecimalPlacesCount)
}
