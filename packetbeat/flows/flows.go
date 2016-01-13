package flows

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/config"
)

type Flows struct {
	worker     *worker
	table      *flowMetaTable
	counterReg *counterReg
}

type Flow struct {
	killed uint32
	id     *FlowID
	stats  flowStats
	ts     time.Time

	prev, next *Flow
}

type FlowStats struct {
	Packets uint64
	Bytes   uint64
}

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
	table map[string]*Flow

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
	head, tail *Flow
}

func NewFlows(pub publisher.Client, config *config.Flows) (*Flows, error) {
	duration := func(s string, d time.Duration) (time.Duration, error) {
		if s == "" {
			return d, nil
		}
		return time.ParseDuration(s)
	}

	defaultTimeout := 30 * time.Second
	timeout, err := duration(config.Timeout, defaultTimeout)
	if err != nil {
		logp.Err("failed to parse flow timeout: %v", err)
		return nil, err
	}

	defaultPeriod := 10 * time.Second
	period, err := duration(config.Period, defaultPeriod)
	if err != nil {
		logp.Err("failed to parse period: %v", err)
		return nil, err
	}

	table := &flowMetaTable{
		table: make(map[flowIDMeta]*flowTable),
	}

	counter := &counterReg{}

	worker, err := newFlowsWorker(pub, table, counter, timeout, period)
	if err != nil {
		logp.Err("failed to configure flows processing intervals: %v", err)
		return nil, err
	}

	return &Flows{
		table:      table,
		worker:     worker,
		counterReg: counter,
	}, nil
}

func (f *Flows) Lock() {
	f.table.Lock()
}

func (f *Flows) Unlock() {
	f.table.Unlock()
}

func (f *Flows) Get(id *FlowID) *Flow {
	if id.flow == nil {
		flow, isNew := f.table.get(id)
		if isNew {
			flow.stats.init(f.counterReg)
		}
		flow.ts = time.Now()
		id.flow = flow
	}
	return id.flow
}

func (f *Flows) Start() {
	f.worker.Start()
}

func (f *Flows) Stop() {
	f.worker.Stop()
}

func (f *Flows) NewInt(name string) (*Int, error) {
	return f.counterReg.newInt(name)
}

func (f *Flows) NewFloat(name string) (*Float, error) {
	return f.counterReg.newFloat(name)
}

func (t *flowMetaTable) get(id *FlowID) (*Flow, bool) {
	sub := t.table[id.flowIDMeta]
	if sub != nil {
		return sub.get(id)
	}

	sub = &flowTable{
		table: make(map[string]*Flow),
		next:  nil,
	}

	t.tables.append(sub)

	return sub.get(id)
}

func (t *flowTable) get(id *FlowID) (*Flow, bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	flow := t.table[string(id.flowID)]
	if flow != nil && flow.alive() {
		return flow, false
	}

	newID := id.Clone()
	flow = &Flow{id: newID}
	t.flows.append(flow)

	t.table[string(newID.flowID)] = flow
	return flow, true
}

func (t *flowTable) remove(f *Flow) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	delete(t.table, string(f.id.flowID))
	t.flows.remove(f)
}

func (f *Flow) kill() {
	atomic.StoreUint32(&f.killed, 1)
}

func (f *Flow) alive() bool {
	return atomic.LoadUint32(&f.killed) == 0
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

func (l *flowList) append(f *Flow) {
	f.prev = l.tail
	f.next = nil

	if l.tail == nil {
		l.head = f
	} else {
		l.tail.next = f
	}
	l.tail = f
}

func (l *flowList) remove(f *Flow) {
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
