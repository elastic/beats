// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"encoding/json"
	"fmt"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
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
	clientsSchema = s.Schema{
		"state":         c.Str("state"),
		"role":          c.Str("role", s.Optional), // cluster role is optional
		"clients":       c.Int("clients"),
		"subscriptions": c.Int("subscriptions"),
		"channels":      c.Int("channels"),
		"messages":      c.Int("total_msgs"),
		"bytes":         c.Int("total_bytes"),
	}
)

func eventMapping(content []byte, r mb.ReporterV2) error {
	var streaming = make(map[string]interface{})
	if err := json.Unmarshal(content, &streaming); err != nil {
		return fmt.Errorf("error in streaming server mapping: %w", err)
	}

	fields, err := clientsSchema.Apply(streaming)
	if err != nil {
		return fmt.Errorf("error parsing Nats streaming server API response: %w", err)
	}

	moduleFields, err := moduleSchema.Apply(streaming)
	if err != nil {
		return fmt.Errorf("error applying module schema: %w", err)
	}
	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    moduleFields,
	}
	if !r.Event(event) {
		return nil
	}
	return nil
}
