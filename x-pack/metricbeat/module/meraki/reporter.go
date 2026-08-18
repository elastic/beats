// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package meraki is a Metricbeat module that contains MetricSets.
package meraki

import (
	"reflect"
	"time"

	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func ReportMetricsForOrganization(reporter mb.ReporterV2, organizationID string, metrics ...[]mapstr.M) {
	for _, metricSlice := range metrics {
		for _, metric := range metricSlice {
			event := mb.Event{ModuleFields: mapstr.M{"organization_id": organizationID}}
			if ts, ok := metric["@timestamp"]; ok {
				tsVal, tsValOk := ts.(string)
				if !tsValOk {
					continue
				}
				t, err := time.Parse(time.RFC3339, tsVal)
				if err == nil {
					// if the timestamp parsing fails, we just fall back to the event time
					// (and leave the additional timestamp in the event for posterity)
					event.Timestamp = t
					delete(metric, "@timestamp")
				}
			}

			for k, v := range metric {
				if isEmpty(v) {
					continue
				}
				event.ModuleFields.Put(k, v)
			}

			reporter.Event(event)
		}
	}
}

func isEmpty(value interface{}) bool {
	// we make use of the fact that all the dashboard API responses utilize
	// pointers for non-string types to filter out empty values from metric events.

	if value == nil {
		return true
	}

	t := reflect.TypeOf(value)

	if t.Kind() == reflect.Ptr {
		return reflect.ValueOf(value).IsNil()
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.String {
		return reflect.ValueOf(value).Len() == 0
	}

	return false
}
