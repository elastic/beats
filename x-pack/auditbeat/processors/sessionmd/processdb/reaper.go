// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"time"
)

const (
	removalCandidateTimeout = 10 * time.Second // remove processes that have been exited longer than this
	orphanTimeout           = 90 * time.Second // remove orphan exit events that have been around longer than this
)

// the reaper logic for removing a process.
// split out to a new function to ease testing.
var removalFuncTimeoutWaiting = func(now, exitTime time.Time) bool {
	return now.Sub(exitTime) < removalCandidateTimeout
}

// the reaper logic for removing an orphaned exit event.
// split out to a new function to ease testing.
var orphanFuncTimeoutWaiting = func(now, exitTime time.Time) bool {
	return now.Sub(exitTime) < orphanTimeout
}

type removalCandidate struct {
	pid       uint32
	exitTime  time.Time
	startTime uint64

	// only used for orphan exit events
	orphanTime time.Time
	exitCode   int32
}

// The reaper will remove exited processes from the DB a short time after they have exited.
// Processes cannot be removed immediately when exiting, as the event enrichment will happen sometime
// afterwards, and will fail if the process is already removed from the DB.
//
// In Linux, exited processes cannot be session leader, process group leader or parent, so if a process has exited,
// it cannot have a relation with any other longer-lived processes. If this processor is ported to other OSs, this
// assumption will need to be revisited.
func (db *DB) startReaper() {
	if db.reaperPeriod > 0 {
		go func(db *DB) {
			ticker := time.NewTicker(db.reaperPeriod)
			defer ticker.Stop()

			for {
				select {
				case <-db.ctx.Done():
					db.logger.Infof("got context done, closing reaper")
					return
				case <-ticker.C:
					db.reapProcs()
				case <-db.stopChan:
					return
				}
			}
		}(db)
	}
}

// run as a separate function to make testing easier
func (db *DB) reapProcs() {
	db.mutex.Lock()
	now := time.Now()
	db.logger.Debugf("REAPER: processes: %d removal candidates: %d", len(db.processes), len(db.removalMap))

	for pid, cand := range db.removalMap {
		// this candidate hasn't reached its timeout, can't be removed yet
		if removalFuncTimeoutWaiting(now, cand.exitTime) {
			continue
		}

		p, ok := db.processes[pid]
		// this represents an orphaned exit event with no matching exec event.
		// in this case, give us a few iterations for us to get the exec, since things can arrive out of order.
		// In our current state, we'll have a lot of orphaned exit events,
		// as we don't track `fork` events.
		if !ok {
			if !orphanFuncTimeoutWaiting(now, cand.orphanTime) {
				db.stats.reapedOrphanExits.Add(1)
				delete(db.removalMap, pid)
			}

			continue
		}
		if p.PIDs.StartTimeNS != cand.startTime {
			// this could happen if the PID has already rolled over and reached this PID again.
			db.logger.Debugf("start times of removal candidate %v differs, not removing (PID had been reused?)", pid)
			continue
		}
		db.stats.reapedProcesses.Add(1)
		delete(db.removalMap, pid)
		delete(db.processes, pid)
		delete(db.entryLeaders, pid)
		delete(db.entryLeaderRelationships, pid)

	}

	// We also need to go through and reap suspect processes.
	// This processor can't rely on any sort of guarantee that we'll get every event,
	// as the audit netlink socket may drop events, and the user may misconfigure
	// the auditd rules so we don't catch every event.
	// as a result, we may need to drop processes that appear orphaned
	var procsToTest []uint32
	if db.reapProcesses {
		for pid, proc := range db.processes {
			// if a process can't be found in procFS, that may mean it's already exited,
			// so we can "safely" remove it after a certain period.
			// however this is still a tad risky, as if the user is running in some kind of
			// container environment where they have access to netlink but not to procfs,
			// we'll remove live processes.
			if proc.procfsLookupFail {
				_, matchingExit := db.removalMap[pid]
				if now.Sub(proc.insertTime) > db.processReapAfter && !matchingExit {
					delete(db.processes, pid)
					// more potential for data loss; if we don't reap these, they can leak, but we may break relationships if a later child PID comes along looking for
					// an entry leader that matches the our orphaned exec event.
					delete(db.entryLeaders, pid)
					delete(db.entryLeaderRelationships, pid)
					db.stats.reapedOrphanProcesses.Add(1)
				}
			} else {
				// be extra cautious with trying to reap processes that we have procfs data for, check to see if the processes are still running first;
				// this is more likely to lead to data loss if running inside a container.
				// In order to check these, we'll need to reach out to /proc, which is more work than I'd rather do while holding a global mutex that's stopping the entire DB.
				// so gather a list now, then check them later
				procsToTest = append(procsToTest, pid)
			}
		}
	}

	db.stats.currentExit.Set(uint64(len(db.removalMap)))
	db.stats.currentProcs.Set(uint64(len(db.processes)))
	db.stats.entryLeaders.Set(uint64(len(db.entryLeaders)))
	db.stats.entryLeaderRelationships.Set(uint64(len(db.entryLeaderRelationships)))

	db.mutex.Unlock()

	// check to make sure that the process still exists.
	if db.reapProcesses && len(procsToTest) > 0 {
		var deadProcs []uint32
		for _, proc := range procsToTest {
			if !db.procfs.ProcessExists(proc) {
				deadProcs = append(deadProcs, proc)
			}
		}

		// now grab mutex again, mark processes we know are dead
		db.mutex.Lock()
		for _, deadProc := range deadProcs {
			if proc, ok := db.processes[deadProc]; ok {
				// set the lookup fail flag, let the rest of the reaper deal with it
				proc.procfsLookupFail = true
				db.processes[deadProc] = proc
			}
		}
		db.mutex.Unlock()
	}
}
