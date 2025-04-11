// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type inputMetrics struct {
	logEventsReceivedTotal       *monitoring.Uint // Number of CloudWatch log events received.
	logGroupsTotal               *monitoring.Uint // Logs collected from number of CloudWatch log groups.
	cloudwatchEventsCreatedTotal *monitoring.Uint // Number of events created from processing logs from CloudWatch.
	apiCallsTotal                *monitoring.Uint // Number of API calls made total.
}

func newInputMetrics(ctx v2.Context) *inputMetrics {
	reg := ctx.MetricsRegistry

	out := &inputMetrics{
		logEventsReceivedTotal:       monitoring.NewUint(reg, "log_events_received_total"),
		logGroupsTotal:               monitoring.NewUint(reg, "log_groups_total"),
		cloudwatchEventsCreatedTotal: monitoring.NewUint(reg, "cloudwatch_events_created_total"),
		apiCallsTotal:                monitoring.NewUint(reg, "api_calls_total"),
	}
	return out
}
