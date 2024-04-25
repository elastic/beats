// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processdb

import (
	"container/heap"
	"time"
)

const (
	reaperInterval = 30 * time.Second // run the reaper process at this interval
	removalTime    = 10 * time.Second // remove processes that have been exited longer than this
)

type removalCandidate struct {
	pid       uint32
	exitTime  time.Time
	startTime uint64
}

type rcHeap []removalCandidate

func (h rcHeap) Len() int {
	return len(h)
}

func (h rcHeap) Less(i, j int) bool {
	return h[i].exitTime.Sub(h[j].exitTime) < 0
}

func (h rcHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *rcHeap) Push(x any) {
	*h = append(*h, x.(removalCandidate))
}

func (h *rcHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
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
		ticker := time.NewTicker(reaperInterval)
		defer ticker.Stop()

		h := &db.removalCandidates
		heap.Init(h)
		for {
			select {
			case <-ticker.C:
				db.mutex.Lock()
				now := time.Now()
				for {
					if len(db.removalCandidates) == 0 {
						break
					}
					v := heap.Pop(h)
					c, ok := v.(removalCandidate)
					if !ok {
						db.logger.Debugf("unexpected item in removal queue: \"%v\"", v)
						continue
					}
					if now.Sub(c.exitTime) < removalTime {
						// this candidate hasn't reached its timeout, put it back on the heap
						// everything else will have a later exit time, so end this run
						heap.Push(h, c)
						break
					}
					p, ok := db.processes[c.pid]
					if !ok {
						db.logger.Debugf("pid %v was candidate for removal, but was already removed", c.pid)
						continue
					}
					if p.PIDs.StartTimeNS != c.startTime {
						// this could happen if the PID has already rolled over and reached this PID again.
						db.logger.Debugf("start times of removal candidate %v differs, not removing (PID had been reused?)", c.pid)
						continue
					}
					delete(db.processes, c.pid)
					delete(db.entryLeaders, c.pid)
					delete(db.entryLeaderRelationships, c.pid)
				}
				db.mutex.Unlock()
			case <-db.stopChan:
				return
			}
		}
	}(db)
}
