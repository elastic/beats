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

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"

	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
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

// Converts uptime from formatted string to seconds
// input: "1y20d22h3m30s", output: 33343410
func convertUptime(uptime string) (seconds int64, err error) {

	var split []string
	var years, days, hours, minutes, secs int64
	if strings.Contains(uptime, "y") {
		split = strings.Split(uptime, "y")
		uptime = split[1]
		years, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid years format in json data")
			return
		}
		seconds += years * 31536000
	}

	if strings.Contains(uptime, "d") {
		split = strings.Split(uptime, "d")
		uptime = split[1]
		days, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid days format in json data")
			return
		}
		seconds += days * 86400
	}

	if strings.Contains(uptime, "h") {
		split = strings.Split(uptime, "h")
		uptime = split[1]
		hours, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid hours format in json data")
			return
		}
		seconds += hours * 3600
	}

	if strings.Contains(uptime, "m") {
		split = strings.Split(uptime, "m")
		uptime = split[1]
		minutes, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid minutes format in json data")
			return
		}
		seconds += minutes * 60
	}

	if strings.Contains(uptime, "s") {
		split = strings.Split(uptime, "s")
		uptime = split[1]
		secs, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid seconds format in json data")
			return
		}
		seconds += secs
	}
	return
}

func eventMapping(r mb.ReporterV2, content []byte) error {
	var event common.MapStr
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		return errors.Wrap(err, "failure parsing Nats stats API response")
	}
	event, err = statsSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying stats schema")
	}

	uptime, err := event.GetValue("uptime")
	if err != nil {
		return errors.Wrap(err, "failure retrieving uptime key")
	}
	uptime, err = convertUptime(uptime.(string))
	if err != nil {
		return errors.Wrap(err, "failure converting uptime from string to integer")
	}
	_, err = event.Put("uptime", uptime)
	if err != nil {
		return errors.Wrap(err, "failure updating uptime key")
	}

	d, err := event.GetValue("http_req_stats")
	if err != nil {
		return errors.Wrap(err, "failure retrieving http_req_stats key")
	}
	httpStats, ok := d.(common.MapStr)
	if !ok {
		return errors.Wrap(err, "failure casting http_req_stats to common.Mapstr")

	}
	err = event.Delete("http_req_stats")
	if err != nil {
		return errors.Wrap(err, "failure deleting http_req_stats key")

	}
	event["http"] = common.MapStr{
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
	cpu, err := event.GetValue("cpu")
	if err != nil {
		return errors.Wrap(err, "failure retrieving cpu key")
	}
	cpuUtil, ok := cpu.(float64)
	if !ok {
		return errors.Wrap(err, "failure casting cpu to float64")
	}
	_, err = event.Put("cpu", cpuUtil/100.0)
	if err != nil {
		return errors.Wrap(err, "failure updating cpu key")
	}
	moduleMetrics, err := moduleSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying module schema")
	}
	r.Event(mb.Event{MetricSetFields: event, ModuleFields: moduleMetrics})
	return nil
}
