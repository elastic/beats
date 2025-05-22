// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/rcrowley/go-metrics"
)

type ipfixMetrics struct {
	FilesOpened    *monitoring.Uint
	FilesClosed    *monitoring.Uint
	ProcessingTime metrics.Sample
}

type netflowMetrics struct {
	discardedEvents *monitoring.Uint
	decodeErrors    *monitoring.Uint
	flows           *monitoring.Uint
	activeSessions  *monitoring.Uint

	ipfix *ipfixMetrics
}

func newInputMetrics(reg *monitoring.Registry) *netflowMetrics {
	if reg == nil {
		return nil
	}

	nm := &netflowMetrics{
		discardedEvents: monitoring.NewUint(reg, "discarded_events_total"),
		flows:           monitoring.NewUint(reg, "flows_total"),
		decodeErrors:    monitoring.NewUint(reg, "decode_errors_total"),
		activeSessions:  monitoring.NewUint(reg, "open_connections"),
		ipfix: &ipfixMetrics{
			FilesOpened:    monitoring.NewUint(reg, "files_opened_total"),
			FilesClosed:    monitoring.NewUint(reg, "files_closed_total"),
			ProcessingTime: metrics.NewUniformSample(1024),
		},
	}

	_ = adapter.NewGoMetrics(reg, "ipfix_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(nm.ipfix.ProcessingTime))

	return nm
}

func (n *netflowMetrics) DiscardedEvents() *monitoring.Uint {
	if n == nil {
		return nil
	}
	return n.discardedEvents
}

func (n *netflowMetrics) FilesOpened() *monitoring.Uint {
	if n == nil || n.ipfix == nil {
		return nil
	}
	return n.ipfix.FilesOpened
}

func (n *netflowMetrics) FilesClosed() *monitoring.Uint {
	if n == nil || n.ipfix == nil {
		return nil
	}
	return n.ipfix.FilesClosed
}

// XXX: this is a hack
func (n *netflowMetrics) Log(path string, what int) {
	if n == nil || n.ipfix == nil {
		return
	}

	if what == 0 {
		n.ipfix.FilesClosed.Inc()
	} else {
		n.ipfix.FilesOpened.Inc()
	}

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
