package flows

import (
	"sync"
	"time"
)

// Table with single produce and single consumer workers.
// Producers will access internal table only and append new tables
// to tail list of known flow tables for consumer to iterate concurrently
// without holding locks for long time. Consumer will never touch the table itself,
// but only iterate the known flow tables.
//
// Note: FlowTables will not be released, as it's assumed different kind of
//       flow tables is limited by network patterns
type flowMetaTable struct {
	sync.Mutex

	table map[flowIDMeta]*flowTable // used by producer worker only

	tables flowTableList

	// TODO: create snapshot of table for concurrent iteration
	// tablesSnapshot flowTableList
}

// Shared flow table.
type flowTable struct {
	mutex sync.Mutex
	table map[string]*biFlow

	// linked list used to delete flows while iterating
	prev, next *flowTable

	flows flowList

	// TODO: create snapshot of table for concurrent iteration
	// flowsSnapshot flowList
}

type flowTableList struct {
	head, tail *flowTable
}

type flowList struct {
	// iterable list of flows for deleting flows during iteration phase
	head, tail *biFlow
}

func (t *flowMetaTable) get(id *FlowID, counter *counterReg) Flow {
	sub := t.table[id.flowIDMeta]
	if sub == nil {
		sub = &flowTable{table: make(map[string]*biFlow)}
		t.table[id.flowIDMeta] = sub
		t.tables.append(sub)
	}
	return sub.get(id, counter)
}

func (t *flowTable) get(id *FlowID, counter *counterReg) Flow {
	ts := time.Now()

	t.mutex.Lock()
	defer t.mutex.Unlock()

	dir := flowDirForward
	bf := t.table[string(id.flowID)]
	if bf == nil || !bf.isAlive() {
		debugf("create new flow")

		bf = newBiFlow(id.rawFlowID.clone(), ts, id.dir)
		t.table[string(bf.id.flowID)] = bf
		t.flows.append(bf)
	} else if bf.dir != id.dir {
		dir = flowDirReversed
	}

	bf.ts = ts
	stats := bf.stats[dir]
	if stats == nil {
		stats = newFlowStats(counter)
		bf.stats[dir] = stats
	}
	return Flow{stats}
}

func (t *flowTable) remove(f *biFlow) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	delete(t.table, string(f.id.flowID))
	t.flows.remove(f)
}

func (l *flowTableList) append(t *flowTable) {
	t.prev = l.tail
	t.next = nil

	if l.tail == nil {
		l.head = t
	} else {
		l.tail.next = t
	}
	l.tail = t
}

func (l *flowList) append(f *biFlow) {
	f.prev = l.tail
	f.next = nil

	if l.tail == nil {
		l.head = f
	} else {
		l.tail.next = f
	}
	l.tail = f
}

func (l *flowList) remove(f *biFlow) {
	if f.next != nil {
		f.next.prev = f.prev
	} else {
		l.tail = f.prev
	}

	if f.prev != nil {
		f.prev.next = f.next
	} else {
		l.head = f.next
	}
}
