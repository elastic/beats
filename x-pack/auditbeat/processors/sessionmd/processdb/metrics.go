// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type Stats struct {
	// number of orphans we have resolved
	resolvedOrphans *monitoring.Uint
	// orphan exit events that were never matched
	reapedOrphans *monitoring.Uint
	// current size of the process map
	currentProcs *monitoring.Uint
	// current size of the exit map
	currentExit *monitoring.Uint
}

func NewStats(reg *monitoring.Registry) *Stats {
	obj := &Stats{
		resolvedOrphans: monitoring.NewUint(reg, "processdb.resolved_orphans"),
		reapedOrphans:   monitoring.NewUint(reg, "processdb.reaped_orphans"),
		currentProcs:    monitoring.NewUint(reg, "processdb.processes"),
		currentExit:     monitoring.NewUint(reg, "processdb.exit_events"),
	}
	return obj
}
