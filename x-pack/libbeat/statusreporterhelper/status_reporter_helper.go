// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// The statusreporterhelper package provides a helper for reporting component state.
// It prevents sending duplicate status updates if the status has not changed.
// It also falls back to standalone mode with debug logs.
package statusreporterhelper

import (
	"sync"

	zap "go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

// StatusReporterHelper helps to report the state of a component via the status package.
type StatusReporterHelper struct {
	current        status.Status
	statusReporter status.StatusReporter

	sync sync.Mutex
}

// New create a new StatusReporterHelper.
func New(statusReporter status.StatusReporter, log *logp.Logger, name string) *StatusReporterHelper {
	rep := &StatusReporterHelper{
		current:        status.Unknown,
		statusReporter: &debugStatusReporter{log: log.WithOptions(zap.AddCallerSkip(2)), name: name},
	}

	if statusReporter != nil {
		rep.statusReporter = statusReporter
	}

	return rep
}

// UpdateStatus updates the status of the component.
func (c *StatusReporterHelper) UpdateStatus(status status.Status, msg string) {
	c.sync.Lock()
	defer c.sync.Unlock()

	// proxy the update only on a state change
	if c.current != status {
		c.current = status
		c.statusReporter.UpdateStatus(c.current, msg)
	}
}

// debugStatusReporter with debugging logs.
// This is typically used when running in standalone mode where injected reporter is nil.
type debugStatusReporter struct {
	log  *logp.Logger
	name string
}

func (n *debugStatusReporter) UpdateStatus(status status.Status, msg string) {
	n.log.Debugf("%s input status updated: status: %s, message: %s", n.name, status, msg)
}
