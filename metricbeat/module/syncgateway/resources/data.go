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

package resources

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/syncgateway"
)

var globalSchema = s.Schema{
	"go_memstats": s.Object{
		"heap": s.Object{
			"alloc":    c.Int("go_memstats_heapalloc"),
			"idle":     c.Int("go_memstats_heapidle"),
			"inuse":    c.Int("go_memstats_heapinuse"),
			"released": c.Int("go_memstats_heapreleased"),
		},
		"pause": s.Object{"ns": c.Int("go_memstats_pausetotalns")},
		"stack": s.Object{
			"inuse": c.Int("go_memstats_stackinuse"),
			"sys":   c.Int("go_memstats_stacksys"),
		},
		"sys": c.Int("go_memstats_sys"),
	},
	"admin_net_bytes": s.Object{
		"recv": c.Int("admin_net_bytes_recv"),
		"sent": c.Int("admin_net_bytes_sent"),
	},
	"error_count":               c.Int("error_count"),
	"goroutines_high_watermark": c.Int("goroutines_high_watermark"),
	"num_goroutines":            c.Int("num_goroutines"),
	"process": s.Object{
		"cpu_percent_utilization": c.Int("process_cpu_percent_utilization"),
		"memory_resident":         c.Int("process_memory_resident"),
	},
	"pub_net": s.Object{
		"recv": s.Object{"bytes": c.Int("pub_net_bytes_recv")},
		"sent": s.Object{"bytes": c.Int("pub_net_bytes_sent")},
	},
	"system_memory_total": c.Int("system_memory_total"),
	"warn_count":          c.Int("warn_count"),
}

func eventMapping(r mb.ReporterV2, content *syncgateway.SgResponse) {
	globalData, _ := globalSchema.Apply(content.Syncgateway.Global.ResourceUtilization)
	r.Event(mb.Event{MetricSetFields: globalData})
}
