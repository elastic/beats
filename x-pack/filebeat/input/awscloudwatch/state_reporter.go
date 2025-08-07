// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

type cwStateReporter struct {
	current  status.Status
	reporter status.StatusReporter

	sync sync.Mutex
}

func newCWStateReporter(ctx v2.Context, log *logp.Logger) *cwStateReporter {
	rep := &cwStateReporter{
		current:  status.Unknown,
		reporter: &debugCWStatusReporter{log: log},
	}

	if ctx.StatusReporter != nil {
		rep.reporter = ctx.StatusReporter
	}

	return rep
}

func (c *cwStateReporter) UpdateStatus(status status.Status, msg string) {
	c.sync.Lock()
	defer c.sync.Unlock()

	// proxy the update only on a state change
	if c.current != status {
		c.current = status
		c.reporter.UpdateStatus(c.current, msg)
	}
}

// debugCWStatusReporter with debugging logs.
// This is typically used when running in standalone mode where injected reporter is nil.
type debugCWStatusReporter struct {
	log *logp.Logger
}

func (n *debugCWStatusReporter) UpdateStatus(status status.Status, msg string) {
	n.log.Debugf("CloudWatch input status updated: status: %s, message: %s", status, msg)
}
