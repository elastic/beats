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
	"time"

	"github.com/pkg/errors"

	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/nats/util"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id": c.Str("server_id"),
		},
	}
	connectionsSchema = s.Schema{
		"name":          c.Str("name"),
		"subscriptions": c.Int("subscriptions"),
		"in": s.Object{
			"messages": c.Int("in_msgs"),
			"bytes":    c.Int("in_bytes"),
		},
		"out": s.Object{
			"messages": c.Int("out_msgs"),
			"bytes":    c.Int("out_bytes"),
		},
		"pending_bytes": c.Int("pending_bytes"),
		"uptime":        c.Str("uptime"),
		"idle_time":     c.Str("idle"),
	}
)

// Connections stores connections related information
type Connections struct {
	Now         time.Time                `json:"now"`
	ServerID    string                   `json:"server_id"`
	Connections []map[string]interface{} `json:"connections,omitempty"`
}

// eventMapping maps a connection to a Metricbeat event using connectionsSchema
func eventMapping(content map[string]interface{}, fieldsSchema s.Schema) (mb.Event, error) {
	fields, err := fieldsSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying connection schema")
	}

	err = util.UpdateDuration(fields, "uptime")
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "failure updating uptime key")
	}

	err = util.UpdateDuration(fields, "idle_time")
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "failure updating idle_time key")
	}

	moduleFields, err := moduleSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying module schema")
	}

	if err != nil {
		return mb.Event{}, errors.Wrap(err, "failure parsing server timestamp")
	}
	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    moduleFields,
	}
	return event, nil
}

// eventsMapping maps per-connection metrics
func eventsMapping(r mb.ReporterV2, content []byte) error {
	var err error
	connections := Connections{}
	if err = json.Unmarshal(content, &connections); err != nil {
		return errors.Wrap(err, "failure parsing NATS connections API response")
	}

	for _, con := range connections.Connections {
		var evt mb.Event
		con["server_id"] = connections.ServerID
		evt, err = eventMapping(con, connectionsSchema)
		if err != nil {
			r.Error(errors.Wrap(err, "error mapping connection event"))
			continue
		}
		evt.Timestamp = connections.Now
		if !r.Event(evt) {
			return nil
		}
	}
	return nil
}
