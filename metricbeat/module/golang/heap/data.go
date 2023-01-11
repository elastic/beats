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

package heap

import (
	"runtime"

	"github.com/elastic/beats/v7/metricbeat/module/golang"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

//Stats contains the memory info that we get from the fetch request
type Stats struct {
	MemStats runtime.MemStats
	Cmdline  []interface{}
}

func eventMapping(stats Stats, m *MetricSet) mapstr.M {
	var event = mapstr.M{
		"cmdline": golang.GetCmdStr(stats.Cmdline),
	}
	//currentNumGC
	ms := &stats.MemStats

	// add heap summary
	event["allocations"] = mapstr.M{
		"mallocs": ms.Mallocs,
		"frees":   ms.Frees,
		"objects": ms.HeapObjects,

		// byte counters
		"total":     ms.TotalAlloc,
		"allocated": ms.HeapAlloc,
		"idle":      ms.HeapIdle,
		"active":    ms.HeapInuse,
	}

	event["system"] = mapstr.M{
		"total":    ms.Sys,
		"obtained": ms.HeapSys,
		"stack":    ms.StackSys,
		"released": ms.HeapReleased,
	}

	// garbage collector summary
	var duration, maxDuration, avgDuration, count uint64
	// collect last gc run stats
	if m.lastNumGC < ms.NumGC {
		delta := ms.NumGC - m.lastNumGC
		start := m.lastNumGC
		if delta > 256 {
			logger.Debug("golang", "Missing %v gc cycles", delta-256)
			start = ms.NumGC - 256
			delta = 256
		}

		end := start + delta
		for i := start; i < end; i++ {
			idx := i % 256
			d := ms.PauseNs[idx]
			count++
			duration += d
			if d > maxDuration {
				maxDuration = d
			}
		}

		avgDuration = duration / count
		m.lastNumGC = ms.NumGC
	}

	event["gc"] = mapstr.M{
		"next_gc_limit": ms.NextGC,
		"total_count":   ms.NumGC,
		"cpu_fraction":  ms.GCCPUFraction,
		"total_pause": mapstr.M{
			"ns": ms.PauseTotalNs,
		},
		"pause": mapstr.M{
			"count": count,
			"sum": mapstr.M{
				"ns": duration,
			},
			"avg": mapstr.M{
				"ns": avgDuration,
			},
			"max": mapstr.M{
				"ns": maxDuration,
			},
		},
	}

	return event

}
