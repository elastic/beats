// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type Stats struct {
	// number of orphans we have resolved, meaning we got the exit event before the exec event.
	resolvedOrphanExits *monitoring.Uint
	// orphan exit events (an exit with no matching exec) that were never matched and later reaped.
	reapedOrphanExits *monitoring.Uint
	// current size of the process map
	currentProcs *monitoring.Uint
	// current size of the exit map
	currentExit *monitoring.Uint
	// number of orphaned (an exec with no matching exst) processes that were removed from the DB by the reaper.
	reapedOrphanProcesses *monitoring.Uint
	// count of times we successfully served a process upstream
	servedProcessCount *monitoring.Uint
	// count of times we could not find a process for the upstream processor
	failedToFindProcessCount *monitoring.Uint
	// count of processes removed from the DB with a matching exit
	reapedProcesses *monitoring.Uint
	// processes where we couldn't find a matching hostfs entry
	procfsLookupFail *monitoring.Uint
	// number of processes marked as session entry leaders
	entryLeaders *monitoring.Uint
	// number of session process relationships
	entryLeaderRelationships *monitoring.Uint
	// number of times we failed to find an entry leader for a process
	entryLeaderLookupFail *monitoring.Uint
}

func NewStats(reg *monitoring.Registry) *Stats {
	obj := &Stats{
		resolvedOrphanExits:      monitoring.NewUint(reg, "resolved_orphan_exits"),
		reapedOrphanExits:        monitoring.NewUint(reg, "reaped_orphan_exits"),
		currentProcs:             monitoring.NewUint(reg, "processes_gauge"),
		currentExit:              monitoring.NewUint(reg, "exit_events_gauge"),
		reapedOrphanProcesses:    monitoring.NewUint(reg, "reaped_orphan_processes"),
		servedProcessCount:       monitoring.NewUint(reg, "served_process_count"),
		failedToFindProcessCount: monitoring.NewUint(reg, "failed_process_lookup_count"),
		reapedProcesses:          monitoring.NewUint(reg, "reaped_processes"),
		procfsLookupFail:         monitoring.NewUint(reg, "procfs_lookup_fail"),
		entryLeaders:             monitoring.NewUint(reg, "entry_leaders_gauge"),
		entryLeaderRelationships: monitoring.NewUint(reg, "entry_leader_relationships_gauge"),
		entryLeaderLookupFail:    monitoring.NewUint(reg, "entry_leader_lookup_fail"),
	}
	return obj
}
