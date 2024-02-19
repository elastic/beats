// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import "github.com/elastic/elastic-agent-libs/monitoring"

type netflowMetrics struct {
	discardedEvents *monitoring.Uint
	decodeErrors    *monitoring.Uint
	flows           *monitoring.Uint
}

func newMetrics(reg *monitoring.Registry) *netflowMetrics {
	return &netflowMetrics{
		discardedEvents: monitoring.NewUint(reg, "discarded_events_total"),
		flows:           monitoring.NewUint(reg, "flows_total"),
		decodeErrors:    monitoring.NewUint(reg, "decode_errors_total"),
	}
}
