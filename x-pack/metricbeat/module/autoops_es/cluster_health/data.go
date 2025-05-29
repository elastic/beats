// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cluster_health

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

var (
	schema = s.Schema{
		// TODO: Throw away cluster name given that we get it from the ClusterInfo
		"cluster_name":                     c.Str("cluster_name", s.Required),
		"status":                           c.Str("status", s.Required),
		"timed_out":                        c.Bool("timed_out", s.IgnoreAllErrors),
		"number_of_nodes":                  c.Int("number_of_nodes", s.IgnoreAllErrors),
		"number_of_data_nodes":             c.Int("number_of_data_nodes", s.IgnoreAllErrors),
		"active_primary_shards":            c.Int("active_primary_shards", s.IgnoreAllErrors),
		"active_shards":                    c.Int("active_shards", s.IgnoreAllErrors),
		"relocating_shards":                c.Int("relocating_shards", s.IgnoreAllErrors),
		"initializing_shards":              c.Int("initializing_shards", s.IgnoreAllErrors),
		"unassigned_shards":                c.Int("unassigned_shards", s.IgnoreAllErrors),
		"delayed_unassigned_shards":        c.Int("delayed_unassigned_shards", s.IgnoreAllErrors),
		"number_of_pending_tasks":          c.Int("number_of_pending_tasks", s.IgnoreAllErrors),
		"number_of_in_flight_fetch":        c.Int("number_of_in_flight_fetch", s.IgnoreAllErrors),
		"task_max_waiting_in_queue_millis": c.Int("task_max_waiting_in_queue_millis", s.IgnoreAllErrors),
		"active_shards_percent_as_number":  c.Int("active_shards_percent_as_number", s.IgnoreAllErrors),
	}
)

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, data *map[string]interface{}) error {
	metricSetFields, err := schema.Apply(*data)

	if err != nil {
		err = fmt.Errorf("failed applying cluster health schema %w", err)
		events.SendErrorEventWithRandomTransactionId(err, info, r, ClusterHealthMetricSet, ClusterHealthPath)
		return err
	}

	r.Event(events.CreateEventWithRandomTransactionId(info, metricSetFields))

	return nil
}
