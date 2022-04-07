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

package route

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id": c.Str("server_id"),
		},
	}
	routesSchema = s.Schema{
		"remote_id":     c.Str("remote_id"),
		"subscriptions": c.Int("subscriptions"),
		"in": s.Object{
			"messages": c.Int("in_msgs"),
			"bytes":    c.Int("in_bytes"),
		},
		"out": s.Object{
			"messages": c.Int("out_msgs"),
			"bytes":    c.Int("out_bytes"),
		},
		"pending_size": c.Int("pending_size"),
		"port":         c.Int("port"),
		"ip":           c.Str("ip"),
	}
)

// Routes stores routes related information
type Routes struct {
	Now      time.Time                `json:"now"`
	ServerID string                   `json:"server_id"`
	Routes   []map[string]interface{} `json:"routes,omitempty"`
}

// eventMapping maps a route to a Metricbeat event using routesSchema
func eventMapping(content map[string]interface{}, fieldsSchema s.Schema) (mb.Event, error) {
	fields, err := fieldsSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying routes schema")
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

// eventsMapping maps per-route metrics
func eventsMapping(r mb.ReporterV2, content []byte) error {
	var err error
	connections := Routes{}
	if err = json.Unmarshal(content, &connections); err != nil {
		return errors.Wrap(err, "failure parsing NATS connections API response")
	}

	for _, con := range connections.Routes {
		var evt mb.Event
		con["server_id"] = connections.ServerID
		evt, err = eventMapping(con, routesSchema)
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
