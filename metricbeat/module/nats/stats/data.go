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

package stats

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/nats/util"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id":   c.Str("server_id"),
			"time": c.Str("now"),
		},
	}
	httpReqStatsSchema = s.Schema{
		"root_uri":   c.Int("/"),
		"connz_uri":  c.Int("/connz"),
		"routez_uri": c.Int("/routez"),
		"subsz_uri":  c.Int("/subsz"),
		"varz_uri":   c.Int("/varz"),
	}
	statsSchema = s.Schema{
		"uptime": c.Str("uptime"),
		"mem": s.Object{
			"bytes": c.Int("mem"),
		},
		"cores":             c.Int("cores"),
		"cpu":               c.Float("cpu"),
		"total_connections": c.Int("total_connections"),
		"remotes":           c.Int("remotes"),
		"in": s.Object{
			"messages": c.Int("in_msgs"),
			"bytes":    c.Int("in_bytes"),
		},
		"out": s.Object{
			"messages": c.Int("out_msgs"),
			"bytes":    c.Int("out_bytes"),
		},
		"slow_consumers": c.Int("slow_consumers"),
		"http_req_stats": c.Dict("http_req_stats", httpReqStatsSchema),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var metricsetMetrics common.MapStr
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		return errors.Wrap(err, "failure parsing Nats stats API response")
	}
	metricsetMetrics, err = statsSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying stats schema")
	}

	err = util.UpdateDuration(metricsetMetrics, "uptime")
	if err != nil {
		return errors.Wrap(err, "failure updating uptime key")
	}

	d, err := metricsetMetrics.GetValue("http_req_stats")
	if err != nil {
		return errors.Wrap(err, "failure retrieving http_req_stats key")
	}
	httpStats, ok := d.(common.MapStr)
	if !ok {
		return errors.Wrap(err, "failure casting http_req_stats to common.Mapstr")

	}
	err = metricsetMetrics.Delete("http_req_stats")
	if err != nil {
		return errors.Wrap(err, "failure deleting http_req_stats key")

	}
	metricsetMetrics["http"] = common.MapStr{
		"req_stats": common.MapStr{
			"uri": common.MapStr{
				"root":   httpStats["root_uri"],
				"connz":  httpStats["connz_uri"],
				"routez": httpStats["routez_uri"],
				"subsz":  httpStats["subsz_uri"],
				"varz":   httpStats["varz_uri"],
			},
		},
	}
	cpu, err := metricsetMetrics.GetValue("cpu")
	if err != nil {
		return errors.Wrap(err, "failure retrieving cpu key")
	}
	cpuUtil, ok := cpu.(float64)
	if !ok {
		return errors.Wrap(err, "failure casting cpu to float64")
	}
	_, err = metricsetMetrics.Put("cpu", cpuUtil/100.0)
	if err != nil {
		return errors.Wrap(err, "failure updating cpu key")
	}
	moduleMetrics, err := moduleSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying module schema")
	}
	timestamp, err := util.GetNatsTimestamp(moduleMetrics)
	if err != nil {
		return errors.Wrap(err, "failure parsing server timestamp")
	}
	evt := mb.Event{
		MetricSetFields: metricsetMetrics,
		ModuleFields:    moduleMetrics,
		Timestamp:       timestamp,
	}
	r.Event(evt)
	return nil
}
