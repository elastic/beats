// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package subscriptions

import (
	"encoding/json"

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
		"cluster": s.Object{
			"id": c.Str("cluster_id"),
		},
	}
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

// eventMapping maps a subscription to a Metricbeat event using subscriptionsSchema
// to parse through each subscription under a presumed channel
func eventMapping(content map[string]interface{}) (mb.Event, error) {
	fields, err := subscriptionsSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying subscription schema")
	}

	moduleFields, err := moduleSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying module schema")
	}

	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    moduleFields,
	}
	return event, nil
}

// Subscription stores subscription related information
type Subscription struct {
	QueueName    string `json:"queue_name"`
	IsDurable    bool   `json:"is_durable"`
	IsOffline    bool   `json:"is_offline"`
	IsStalled    bool   `json:"is_stalled"`
	PendingCount int64  `json:"pending_count"`
	LastSent     int64  `json:"last_sent"`
}

// Channel stores channel related information
type Channel struct {
	Name    string `json:"name"`
	Msgs    int64  `json:"msgs"`
	Bytes   int64  `json:"bytes"`
	LastSeq int64  `json:"last_seq"`
	// Subscriptions []Subscription `json:"subscriptions,omitempty"`
	Subscriptions []map[string]interface{} `json:"subscriptions,omitempty"`
}

// Channels stores channels related information
type Channels struct {
	ClusterID string    `json:"cluster_id"`
	ServerID  string    `json:"server_id"`
	Limit     uint64    `json:"limit"`
	Total     uint64    `json:"total"`
	Channels  []Channel `json:"channels,omitempty"`
}

// eventsMapping maps the top-level channel metrics AND also per-channel metrics AND subscriptions
func eventsMapping(content []byte, r mb.ReporterV2) error {
	var err error
	channels := Channels{}
	if err = json.Unmarshal(content, &channels); err != nil {
		return errors.Wrap(err, "error unmarshaling Nats streaming channels detailed response to JSON")
	}

	for _, ch := range channels.Channels {
		for _, sub := range ch.Subscriptions {
			var evt mb.Event
			sub["channel"] = ch.Name
			sub["server_id"] = channels.ServerID
			sub["cluster_id"] = channels.ClusterID
			evt, err = eventMapping(sub)
			if err != nil {
				r.Error(errors.Wrap(err, "error mapping subscription event"))
				continue
			}

			if !r.Event(evt) {
				return nil
			}
		}
	}
	return nil
}
