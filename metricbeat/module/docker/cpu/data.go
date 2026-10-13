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
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
)

func eventsMapping(r mb.ReporterV2, cpuStatsList []CPUStats) {
	for i := range cpuStatsList {
		eventMapping(r, &cpuStatsList[i])
	}
}

func eventMapping(r mb.ReporterV2, stats *CPUStats) {
	kernel := mapstr.M{"ticks": stats.UsageInKernelmode}
	addPercentageFields(kernel, stats.UsageInKernelmodePercentage, stats.UsageInKernelmodePercentageNormalized)

	user := mapstr.M{"ticks": stats.UsageInUsermode}
	addPercentageFields(user, stats.UsageInUsermodePercentage, stats.UsageInUsermodePercentageNormalized)

	system := mapstr.M{"ticks": stats.SystemUsage}
	addPercentageFields(system, stats.SystemUsagePercentage, stats.SystemUsagePercentageNormalized)

	fields := mapstr.M{
		"core":   stats.PerCPUUsage,
		"kernel": kernel,
		"user":   user,
		"system": system,
	}
	total := mapstr.M{}
	addPercentageFields(total, stats.TotalUsage, stats.TotalUsageNormalized)
	if len(total) != 0 {
		fields["total"] = total
	}

	rootFields := stats.Container.ToMapStr()
	// Add container ECS fields
	if stats.TotalUsageNormalized.Exists() {
		_, _ = rootFields.Put("container.cpu.usage", stats.TotalUsageNormalized.ValueOr(0))
	}

	r.Event(mb.Event{
		RootFields:      rootFields,
		MetricSetFields: fields,
	})
}

func addPercentageFields(fields mapstr.M, percentage, normalized opt.Float) {
	if percentage.Exists() {
		fields["pct"] = percentage.ValueOr(0)
	}
	if normalized.Exists() {
		fields["norm"] = mapstr.M{"pct": normalized.ValueOr(0)}
	}
}
