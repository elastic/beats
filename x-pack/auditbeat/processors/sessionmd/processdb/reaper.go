// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"time"
)

const (
	removalTimeout     = 10 * time.Second // remove processes that have been exited longer than this
	exitRemoveAttempts = 2                // Number of times to run the reaper before we remove an orphaned exit event
)

// the reaper logic for removing a process.
// split out to a new function to ease testing.
var functionTimeoutReached = func(now, exitTime time.Time) bool {
	return now.Sub(exitTime) < removalTimeout
}

type removalCandidate struct {
	pid       uint32
	exitTime  time.Time
	startTime uint64

	// only used for orphan exit events
	removeAttempt uint32
	exitCode      int32
}

// The reaper will remove exited processes from the DB a short time after they have exited.
// Processes cannot be removed immediately when exiting, as the event enrichment will happen sometime
// afterwards, and will fail if the process is already removed from the DB.
//
// In Linux, exited processes cannot be session leader, process group leader or parent, so if a process has exited,
// it cannot have a relation with any other longer-lived processes. If this processor is ported to other OSs, this
// assumption will need to be revisited.
func (db *DB) startReaper() {
	go func(db *DB) {
		ticker := time.NewTicker(db.reaperPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !db.skipReaper {
					db.reapProcs()
				}
			case <-db.stopChan:
				return
			}
		}
	}(db)
}

// run as a separate function to make testing easier
func (db *DB) reapProcs() {
	now := time.Now()
	db.mutex.Lock()
	db.logger.Debugf("REAPER: processes: %d removal candidates: %d", len(db.processes), len(db.removalMap))
	db.stats.currentExit.Set(uint64(len(db.removalMap)))
	db.stats.currentProcs.Set(uint64(len(db.processes)))

	for pid, cand := range db.removalMap {
		if functionTimeoutReached(now, cand.exitTime) {
			// this candidate hasn't reached its timeout
			continue
		}

		p, ok := db.processes[pid]
		if !ok {
			// this represents an orphaned exit event with no matching exec event.
			// in this case, give us a few iterations for us to get the exec, since things can arrive out of order.
			if cand.removeAttempt < exitRemoveAttempts {
				cand.removeAttempt += 1
				db.removalMap[pid] = cand
			} else {
				// in our current state, we'll have a lot of orphaned exit events,
				// as we don't track `fork` events.
				db.logger.Debugf("reaping orphaned exit event for pid %d", pid)
				db.stats.reapedOrphans.Add(1)
				delete(db.removalMap, pid)
			}

			db.logger.Debugf("pid %v was candidate for removal, but was not found", pid)
			continue
		}
		if p.PIDs.StartTimeNS != cand.startTime {
			// this could happen if the PID has already rolled over and reached this PID again.
			db.logger.Debugf("start times of removal candidate %v differs, not removing (PID had been reused?)", pid)
			continue
		}
		delete(db.removalMap, pid)
		delete(db.processes, pid)
		delete(db.entryLeaders, pid)
		delete(db.entryLeaderRelationships, pid)
	}
	db.mutex.Unlock()
}
