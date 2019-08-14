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

package subscriptions

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	subscriptionsSchema = s.Schema{
		"id":        c.Str("client_id"),
		"channel":   c.Str("channel"),                // this is a computed field added AFTER schema.Apply
		"queue":     c.Str("queue_name", s.Optional), // not all STAN channels report as NATS queue associated
		"last_sent": c.Int("last_sent"),
		"offline":   c.Bool("is_offline"),
		"stalled":   c.Bool("is_stalled"),
		"pending":   c.Int("pending_count"),
	}
)

// subscriptionsSchema used to parse through each subscription
// under a presumed channel
func eventMapping(content map[string]interface{}) (mb.Event, error) {
	fields, err := subscriptionsSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "failure applying subscription schema")
	}
	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    common.MapStr{},
	}
	return event, nil
}

type Subscription struct {
	QueueName    string `json:"queue_name"`
	IsDurable    bool   `json:"is_durable"`
	IsOffline    bool   `json:"is_offline"`
	IsStalled    bool   `json:"is_stalled"`
	PendingCount int64  `json:"pending_count"`
	LastSent     int64  `json:"last_sent"`
}
type Channel struct {
	Name    string `json:"name"`
	Msgs    int64  `json:"msgs"`
	Bytes   int64  `json:"bytes"`
	LastSeq int64  `json:"last_seq"`
	// Subscriptions []Subscription `json:"subscriptions,omitempty"`
	Subscriptions []map[string]interface{} `json:"subscriptions,omitempty"`
}

type Channels struct {
	ClusterID string    `json:"cluster_id"`
	Limit     uint64    `json:"limit"`
	Total     uint64    `json:"total"`
	Channels  []Channel `json:"channels,omitempty"`
}

// eventsMapping maps the top-level channel metrics AND also per-channel metrics AND subscriptions
func eventsMapping(content []byte, r mb.ReporterV2) error {
	var err error
	channels := Channels{}
	if err = json.Unmarshal(content, &channels); err != nil {
		return errors.Wrap(err, "failure unmarshaling Nats streaming channels detailed response to JSON")
	}

	for _, ch := range channels.Channels {
		for _, sub := range ch.Subscriptions {
			var evt mb.Event
			sub["channel"] = ch.Name
			evt, err = eventMapping(sub)
			if err != nil {
				r.Error(errors.Wrap(err, "error mapping subscription event"))
			}

			if !r.Event(evt) {
				r.Error(errors.New("Error emitting event"))
			}
		}
	}
	return nil
}
