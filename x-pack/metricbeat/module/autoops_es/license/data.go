// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package license

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	utils "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var (
	schema = s.Schema{
		"license": c.Dict("license", s.Schema{
			"status":                c.Str("status", s.Required),
			"uid":                   c.Str("uid", s.Required),
			"type":                  c.Str("type", s.Required),
			"issue_date":            c.Ifc("issue_date", s.Required),
			"issue_date_in_millis":  c.Int("issue_date_in_millis", s.Optional),
			"expiry_date":           c.Ifc("expiry_date", s.Optional),
			"expiry_date_in_millis": c.Int("expiry_date_in_millis", s.Optional),
			"max_nodes":             c.Ifc("max_nodes", s.Optional),
			"max_resource_units":    c.Ifc("max_resource_units", s.Optional),
			"issued_to":             c.Str("issued_to", s.Required),
			"issuer":                c.Str("issuer", s.Required),
			"start_date_in_millis":  c.Int("start_date_in_millis", s.Optional),
		}),
	}
)

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, data *map[string]any) error {
	metricSetFields, err := schema.Apply(*data)

	if err != nil {
		err = fmt.Errorf("failed applying license schema: %w", err)
		events.SendErrorEventWithRandomTransactionId(err, info, r, LicenseMetricsSet, LicensePath)
		return err
	}

	r.Event(events.CreateEventWithRandomTransactionId(info, metricSetFields))

	return nil
}
