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
	resolvedOrphanExits *monitoring.Uint
	// orphan exit events that were never matched
	reapedOrphanExits *monitoring.Uint
	// current size of the process map
	currentProcs *monitoring.Uint
	// current size of the exit map
	currentExit *monitoring.Uint
	// number of processes that were removed by the reaper
	reapedOrphanProcesses *monitoring.Uint
	// count of times we successfully served a process upstream
	servedProcessCount *monitoring.Uint
	// count of times we could not find a process for upstream
	failedToFindProcessCount *monitoring.Uint
	// count of processes removed after exits are resolved
	reapedProcesses *monitoring.Uint
	// processes where we couldn't find a matching hostfs entry
	procfsLookupFail *monitoring.Uint
}

func NewStats(reg *monitoring.Registry) *Stats {
	obj := &Stats{
		resolvedOrphanExits:      monitoring.NewUint(reg, "processdb.resolved_orphan_exits"),
		reapedOrphanExits:        monitoring.NewUint(reg, "processdb.reaped_orphan_exits"),
		currentProcs:             monitoring.NewUint(reg, "processdb.processes"),
		currentExit:              monitoring.NewUint(reg, "processdb.exit_events"),
		reapedOrphanProcesses:    monitoring.NewUint(reg, "processdb.reaped_orphan_processes"),
		servedProcessCount:       monitoring.NewUint(reg, "processdb.served_process_count"),
		failedToFindProcessCount: monitoring.NewUint(reg, "processdb.failed_process_lookup_count"),
		reapedProcesses:          monitoring.NewUint(reg, "processdb.reaped_processes"),
		procfsLookupFail:         monitoring.NewUint(reg, "processdb.procfs_lookup_fail"),
	}
	return obj
}
