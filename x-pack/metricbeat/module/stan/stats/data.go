// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	clientsSchema = s.Schema{
		"cluster_id":    c.Str("cluster_id"),
		"server_id":     c.Str("server_id"),
		"state":         c.Str("state"),
		"role":          c.Str("role", s.Optional), // cluster role is optional
		"clients":       c.Int("clients"),
		"subscriptions": c.Int("subscriptions"),
		"channels":      c.Int("channels"),
		"msgs":          c.Int("total_msgs"),
		"bytes":         c.Int("total_bytes"),
	}
)

func eventMapping(content []byte, r mb.ReporterV2) error {
	var streaming = make(map[string]interface{})
	if err := json.Unmarshal(content, &streaming); err != nil {
		return errors.Wrap(err, "error in streaming server mapping")
	}

	fields, err := clientsSchema.Apply(streaming)
	if err != nil {
		return errors.Wrap(err, "failure parsing Nats streaming server API response")
	}

	moduleFields := common.MapStr{}
	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    moduleFields,
	}
	if !r.Event(event) {
		err := errors.New("Failed to report event")
		r.Error(err)
		return err
	}
	return nil
}
