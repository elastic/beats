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

package connection

import (
	"encoding/json"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"name":        c.Str("name"),
		"vhost":       c.Str("vhost"),
		"user":        c.Str("user"),
		"node":        c.Str("node"),
		"channels":    c.Int("channels"),
		"channel_max": c.Int("channel_max"),
		"frame_max":   c.Int("frame_max"),
		"type":        c.Str("type"),
		"packet_count": s.Object{
			"sent":     c.Int("send_cnt"),
			"received": c.Int("recv_cnt"),
			"pending":  c.Int("send_pend"),
		},
		"octet_count": s.Object{
			"sent":     c.Int("send_oct"),
			"received": c.Int("recv_oct"),
		},
		"host": c.Str("host"),
		"port": c.Int("port"),
		"peer": s.Object{
			"host": c.Str("peer_host"),
			"port": c.Int("peer_port"),
		},
	}
)

func eventsMapping(content []byte, r mb.ReporterV2) {
	var connections []map[string]interface{}
	err := json.Unmarshal(content, &connections)
	if err != nil {
		logp.Err("Error: %+v", err)
		r.Error(err)
		return
	}

	var errors multierror.Errors
	for _, node := range connections {
		err := eventMapping(node, r)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		r.Error(errors.Err())
	}
}

func eventMapping(connection map[string]interface{}, r mb.ReporterV2) error {
	fields, err := schema.Apply(connection)

	rootFields := common.MapStr{}
	if v, err := fields.GetValue("user"); err == nil {
		rootFields.Put("user.name", v)
		fields.Delete("user")
	}

	moduleFields := common.MapStr{}
	if v, err := fields.GetValue("vhost"); err == nil {
		moduleFields.Put("vhost", v)
		fields.Delete("vhost")
	}

	if v, err := fields.GetValue("node"); err == nil {
		moduleFields.Put("node.name", v)
		fields.Delete("node")
	}

	event := mb.Event{
		MetricSetFields: fields,
		RootFields:      rootFields,
		ModuleFields:    moduleFields,
	}
	r.Event(event)
	return err
}
