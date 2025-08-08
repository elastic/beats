// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package statereporter

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

// EnhancedStatusReporter reports the state of a component via the status package,
// with enhancements.
// Enhancements:
// - it disallows repeats of status updates unless there is a status change
// - it also supports a built-in debug status reporter for standalone (Filebeat-only) mode
type EnhancedStatusReporter struct {
	current        status.Status
	statusReporter status.StatusReporter

	sync sync.Mutex
}

// New create a new StateReporter.
func New(statusReporter status.StatusReporter, log *logp.Logger) *EnhancedStatusReporter {
	rep := &EnhancedStatusReporter{
		current:        status.Unknown,
		statusReporter: &debugStatusReporter{log: log},
	}

	if statusReporter != nil {
		rep.statusReporter = statusReporter
	}

	return rep
}

// UpdateStatus updates the status of the component.
func (c *EnhancedStatusReporter) UpdateStatus(status status.Status, msg string) {
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
	log *logp.Logger
}

func (n *debugStatusReporter) UpdateStatus(status status.Status, msg string) {
	n.log.Debugf("Input status updated: status: %s, message: %s", status, msg)
}
