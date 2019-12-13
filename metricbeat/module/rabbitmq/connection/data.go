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

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"name": c.Str("name"),
		"client_provided": s.Object{
			"name": c.Str("client_properties.connection_name"),
		},
		"vhost":       c.Str("vhost", s.Required),
		"user":        c.Str("user", s.Required),
		"node":        c.Str("node", s.Required),
		"state":       c.Str("state"),
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

func eventsMapping(content []byte, r mb.ReporterV2, m *MetricSet) error {
	var connections []map[string]interface{}
	err := json.Unmarshal(content, &connections)
	if err != nil {
		return errors.Wrap(err, "error in unmarshal")
	}

	for _, node := range connections {
		evt, err := eventMapping(node)
		if err != nil {
			m.Logger().Errorf("error in mapping: %s", err)
			r.Error(err)
			continue
		}

		if !r.Event(evt) {
			return nil
		}
	}
	return nil
}

func eventMapping(connection map[string]interface{}) (mb.Event, error) {
	fields, err := schema.Apply(connection, s.FailOnRequired)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying schema")
	}

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
	return event, nil
}
