package flows

type Flows struct {
	table flowTable
}

type Flow struct {
	id    *FlowID
	Stats FlowStats
}

type FlowStats struct {
	Packets uint64
	Bytes   uint64
}

type flowTable struct {
	table map[flowIDMeta]map[string]*Flow
}

func NewFlows() *Flows {
	return &Flows{
		table: flowTable{
			table: make(map[flowIDMeta]map[string]*Flow),
		},
	}
}

func (f *Flows) Get(id *FlowID) *Flow {
	if id.flow == nil {
		id.flow = f.table.get(id)
	}
	return id.flow
}

func (t *flowTable) lookup(id *FlowID) (*Flow, bool) {
	sub := t.table[id.flowIDMeta]
	if sub == nil {
		sub = make(map[string]*Flow)
		t.table[id.flowIDMeta] = sub
	}

	flow, ok := sub[string(id.flowID)]
	return flow, ok
}

func (t *flowTable) get(id *FlowID) *Flow {
	sub := t.table[id.flowIDMeta]
	if sub == nil {
		sub = make(map[string]*Flow)
		t.table[id.flowIDMeta] = sub
	}

	if flow, ok := sub[string(id.flowID)]; ok {
		return flow
	}

	return t.newFlow(sub, id)
}

func (t *flowTable) newFlow(tbl map[string]*Flow, id *FlowID) *Flow {
	newID := id.Clone()
	f := &Flow{id: newID}
	tbl[string(newID.flowID)] = f
	return f
}
