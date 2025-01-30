// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import "github.com/elastic/elastic-agent-libs/monitoring"

type netflowMetrics struct {
	discardedEvents *monitoring.Uint
	decodeErrors    *monitoring.Uint
	flows           *monitoring.Uint
	activeSessions  *monitoring.Uint
}

func newInputMetrics(reg *monitoring.Registry) *netflowMetrics {
	if reg == nil {
		return nil
	}

	return &netflowMetrics{
		discardedEvents: monitoring.NewUint(reg, "discarded_events_total"),
		flows:           monitoring.NewUint(reg, "flows_total"),
		decodeErrors:    monitoring.NewUint(reg, "decode_errors_total"),
		activeSessions:  monitoring.NewUint(reg, "open_connections"),
	}
}

func (n *netflowMetrics) DiscardedEvents() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.discardedEvents
}

func (n *netflowMetrics) DecodeErrors() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.decodeErrors
}

func (n *netflowMetrics) Flows() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.flows
}

func (n *netflowMetrics) ActiveSessions() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.activeSessions
}
