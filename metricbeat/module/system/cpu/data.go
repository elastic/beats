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

package cpu

import (
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/cpu"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// CPU metrics are highly OS-specific, so we need to build the event per-OS
func getPlatformCPUMetrics(sample *cpu.Metrics, selectors []string, event common.MapStr) {
	for _, metric := range selectors {
		switch strings.ToLower(metric) {
		case percentages:
			pct := sample.Percentages()
			event.Put("user.pct", pct.User)
			event.Put("system.pct", pct.System)
			event.Put("idle.pct", pct.Idle)
			event.Put("total.pct", pct.Total)

			if runtime.GOOS != "windows" {
				event.Put("nice.pct", pct.Nice)
			}
			if runtime.GOOS == "linux" || runtime.GOOS == "openbsd" {
				event.Put("irq.pct", pct.IRQ)
			}
			if runtime.GOOS == "linux" || runtime.GOOS == "aix" {
				event.Put("iowait.pct", pct.IOWait)
			}
			if runtime.GOOS == "linux" {
				event.Put("softirq.pct", pct.SoftIRQ)
				event.Put("steal.pct", pct.Steal)
			}
		case normalizedPercentages:
			normalizedPct := sample.NormalizedPercentages()
			event.Put("user.norm.pct", normalizedPct.User)
			event.Put("system.norm.pct", normalizedPct.System)
			event.Put("idle.norm.pct", normalizedPct.Idle)
			event.Put("total.norm.pct", normalizedPct.Total)

			if runtime.GOOS != "windows" {
				event.Put("nice.norm.pct", normalizedPct.Nice)
			}
			if runtime.GOOS == "linux" || runtime.GOOS == "openbsd" {
				event.Put("irq.norm.pct", normalizedPct.IRQ)
			}
			if runtime.GOOS == "linux" || runtime.GOOS == "aix" {
				event.Put("iowait.norm.pct", normalizedPct.IOWait)
			}
			if runtime.GOOS == "linux" {
				event.Put("softirq.norm.pct", normalizedPct.SoftIRQ)
				event.Put("steal.norm.pct", normalizedPct.Steal)
			}
		case ticks:
			ticks := sample.Ticks()
			event.Put("user.ticks", ticks.User)
			event.Put("system.ticks", ticks.System)
			event.Put("idle.ticks", ticks.Idle)

			if runtime.GOOS != "windows" {
				event.Put("nice.ticks", ticks.Nice)
			}
			if runtime.GOOS == "linux" || runtime.GOOS == "openbsd" {
				event.Put("irq.ticks", ticks.IRQ)
			}
			if runtime.GOOS == "linux" || runtime.GOOS == "aix" {
				event.Put("iowait.ticks", ticks.IOWait)
			}
			if runtime.GOOS == "linux" {
				event.Put("softirq.ticks", ticks.SoftIRQ)
				event.Put("steal.ticks", ticks.Steal)
			}
		}
	}
}

// gather CPU metrics
func collectCPUMetrics(selectors []string, sample *cpu.Metrics) mb.Event {
	event := common.MapStr{"cores": runtime.NumCPU()}
	getPlatformCPUMetrics(sample, selectors, event)

	//generate the host fields here, since we don't want users disabling it.
	normalizedPct := sample.NormalizedPercentages()
	hostFields := common.MapStr{}
	hostFields.Put("host.cpu.usage", normalizedPct.Total)

	return mb.Event{
		RootFields:      hostFields,
		MetricSetFields: event,
	}
}
