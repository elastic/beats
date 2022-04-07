// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"github.com/elastic/beats/v8/libbeat/monitoring"
)

type inputMetrics struct {
	id     string               // Input ID.
	parent *monitoring.Registry // Parent registry holding this input's ID as a key.

	logEventsReceivedTotal       *monitoring.Uint // Number of CloudWatch log events received.
	logGroupsTotal               *monitoring.Uint // Logs collected from number of CloudWatch log groups.
	cloudwatchEventsCreatedTotal *monitoring.Uint // Number of events created from processing logs from CloudWatch.
	apiCallsTotal                *monitoring.Uint // Number of API calls made total.
}

// Close removes the metrics from the registry.
func (m *inputMetrics) Close() {
	m.parent.Remove(m.id)
}

func newInputMetrics(parent *monitoring.Registry, id string) *inputMetrics {
	reg := parent.NewRegistry(id)
	monitoring.NewString(reg, "input").Set(inputName)
	monitoring.NewString(reg, "id").Set(id)
	out := &inputMetrics{
		id:                           id,
		parent:                       reg,
		logEventsReceivedTotal:       monitoring.NewUint(reg, "log_events_received_total"),
		logGroupsTotal:               monitoring.NewUint(reg, "log_groups_total"),
		cloudwatchEventsCreatedTotal: monitoring.NewUint(reg, "cloudwatch_events_created_total"),
		apiCallsTotal:                monitoring.NewUint(reg, "api_calls_total"),
	}
	return out
}
