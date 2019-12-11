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

package stat

import (
	"reflect"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/haproxy"
)

var (
	schema = s.Schema{
		"status":         c.Str("Status"),
		"weight":         c.Int("Weight", s.Optional),
		"downtime":       c.Int("Downtime", s.Optional),
		"component_type": c.Int("Type"),
		"process_id":     c.Int("Pid"),
		"service_name":   c.Str("SvName"),
		"in.bytes":       c.Int("Bin"),
		"out.bytes":      c.Int("Bout"),
		"last_change":    c.Int("Lastchg", s.Optional),
		"throttle.pct":   c.Int("Throttle", s.Optional),
		"selected.total": c.Int("Lbtot", s.Optional),
		"tracked.id":     c.Int("Tracked", s.Optional),

		"connection": s.Object{
			"total":    c.Int("Stot"),
			"retried":  c.Int("Wretr", s.Optional),
			"time.avg": c.Int("Ctime", s.Optional),
		},

		"request": s.Object{
			"denied":            c.Int("Dreq", s.Optional),
			"queued.current":    c.Int("Qcur", s.Optional),
			"queued.max":        c.Int("Qmax", s.Optional),
			"errors":            c.Int("Ereq", s.Optional),
			"redispatched":      c.Int("Wredis", s.Optional),
			"connection.errors": c.Int("Econ", s.Optional),
			"rate": s.Object{
				"value": c.Int("ReqRate", s.Optional),
				"max":   c.Int("ReqRateMax", s.Optional),
			},
			"total": c.Int("ReqTot", s.Optional),
		},

		"response": s.Object{
			"errors":   c.Int("Eresp", s.Optional),
			"time.avg": c.Int("Rtime", s.Optional),
			"denied":   c.Int("Dresp"),
			"http": s.Object{
				"1xx":   c.Int("Hrsp1xx", s.Optional),
				"2xx":   c.Int("Hrsp2xx", s.Optional),
				"3xx":   c.Int("Hrsp3xx", s.Optional),
				"4xx":   c.Int("Hrsp4xx", s.Optional),
				"5xx":   c.Int("Hrsp5xx", s.Optional),
				"other": c.Int("HrspOther", s.Optional),
			},
		},

		"session": s.Object{
			"current": c.Int("Scur"),
			"max":     c.Int("Smax"),
			"limit":   c.Int("Slim", s.Optional),
			"rate": s.Object{
				"value": c.Int("Rate", s.Optional),
				"limit": c.Int("RateLim", s.Optional),
				"max":   c.Int("RateMax", s.Optional),
			},
		},

		"check": s.Object{
			"status":      c.Str("CheckStatus"),
			"code":        c.Int("CheckCode", s.Optional),
			"duration":    c.Int("CheckDuration", s.Optional),
			"health.last": c.Str("LastChk"),
			"health.fail": c.Int("Hanafail", s.Optional),
			"agent.last":  c.Str("LastAgt"),
			"failed":      c.Int("ChkFail", s.Optional),
			"down":        c.Int("ChkDown", s.Optional),
		},

		"client.aborted": c.Int("CliAbrt", s.Optional),

		"server": s.Object{
			"id":      c.Int("Sid"),
			"aborted": c.Int("SrvAbrt", s.Optional),
			"active":  c.Int("Act", s.Optional),
			"backup":  c.Int("Bck", s.Optional),
		},

		"compressor": s.Object{
			"in.bytes":       c.Int("CompIn", s.Optional),
			"out.bytes":      c.Int("CompOut", s.Optional),
			"bypassed.bytes": c.Int("CompByp", s.Optional),
			"response.bytes": c.Int("CompRsp", s.Optional),
		},

		"proxy": s.Object{
			"id":   c.Int("Iid"),
			"name": c.Str("PxName"),
		},

		"queue": s.Object{
			"time.avg": c.Int("Qtime", s.Optional),
			"limit":    c.Int("Qlimit", s.Optional),
		},
	}
)

// Map data to MapStr.
func eventMapping(info []*haproxy.Stat, r mb.ReporterV2) {
	for _, evt := range info {
		st := reflect.ValueOf(evt).Elem()
		typeOfT := st.Type()
		source := map[string]interface{}{}

		for i := 0; i < st.NumField(); i++ {
			f := st.Field(i)
			source[typeOfT.Field(i).Name] = f.Interface()
		}

		fields, _ := schema.Apply(source)
		event := mb.Event{
			RootFields: common.MapStr{},
		}

		if processID, err := fields.GetValue("process_id"); err == nil {
			event.RootFields.Put("process.pid", processID)
			fields.Delete("process_id")
		}

		event.MetricSetFields = fields
		r.Event(event)
	}
}
