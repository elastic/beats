// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package channels

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
	channelSchema = s.Schema{
		"name":      c.Str("name"),
		"messages":  c.Int("msgs"),
		"bytes":     c.Int("bytes"),
		"first_seq": c.Int("first_seq"),
		"last_seq":  c.Int("last_seq"),
		"depth":     c.Int("depth", s.Optional), // aggregated by the module
	}
)

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
	Name          string         `json:"name"`
	Msgs          int64          `json:"msgs"`
	Bytes         int64          `json:"bytes"`
	FirstSeq      int64          `json:"first_seq"`
	LastSeq       int64          `json:"last_seq"`
	Subscriptions []Subscription `json:"subscriptions,omitempty"`
}

// Channels stores channels related information
type Channels struct {
	ClusterID string    `json:"cluster_id"`
	ServerID  string    `json:"server_id"`
	Limit     uint64    `json:"limit"`
	Total     uint64    `json:"total"`
	Channels  []Channel `json:"channels,omitempty"`
}

// eventMapping maps a channel to a Metricbeat event
func eventMapping(content map[string]interface{}) (mb.Event, error) {
	fields, err := channelSchema.Apply(content)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error applying channels schema")
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

// eventsMapping maps the top-level channel metrics AND also per-channel metrics AND subscriptions
func eventsMapping(content []byte, r mb.ReporterV2) error {
	channelsIn := Channels{}
	if err := json.Unmarshal(content, &channelsIn); err != nil {
		return errors.Wrap(err, "error unmarshaling Nats streaming channels response to JSON")
	}

	for _, ch := range channelsIn.Channels {
		var evt mb.Event
		var err error
		var maxSubSeq int64
		for _, sub := range ch.Subscriptions {
			if sub.LastSent > maxSubSeq {
				maxSubSeq = sub.LastSent
			}
		}
		chWrapper := map[string]interface{}{
			"cluster_id": channelsIn.ClusterID,
			"server_id":  channelsIn.ServerID,
			"name":       ch.Name,
			"msgs":       ch.Msgs,
			"bytes":      ch.Bytes,
			"first_seq":  ch.FirstSeq,
			"last_seq":   ch.LastSeq,
			"depth":      ch.LastSeq - maxSubSeq, // queue depth is known channel seq number - maximum consumed by subscribers
		}

		if evt, err = eventMapping(chWrapper); err != nil {
			r.Error(errors.Wrap(err, "error mapping channel to its schema"))
			continue
		}
		if !r.Event(evt) {
			return nil
		}
	}

	return nil
}
