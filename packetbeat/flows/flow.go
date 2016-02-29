package flows

import (
	"sync/atomic"
	"time"
)

type biFlow struct {
	id       rawFlowID
	killed   uint32
	createTS time.Time
	ts       time.Time

	dir        flowDirection
	stats      [2]*flowStats
	prev, next *biFlow
}

type Flow struct {
	stats *flowStats
}

func newBiFlow(id rawFlowID, ts time.Time, dir flowDirection) *biFlow {
	return &biFlow{
		id:       id,
		ts:       ts,
		createTS: ts,
		dir:      dir,
	}
}

func (f *biFlow) kill() {
	atomic.StoreUint32(&f.killed, 1)
}

func (f *biFlow) isAlive() bool {
	return atomic.LoadUint32(&f.killed) == 0
}
