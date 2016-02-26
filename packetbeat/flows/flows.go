package flows

import (
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/publish"
)

type Flows struct {
	worker     *worker
	table      *flowMetaTable
	counterReg *counterReg
}

var debugf = logp.MakeDebug("flows")

const (
	defaultTimeout = 30 * time.Second
	defaultPeriod  = 10 * time.Second
)

func NewFlows(pub publish.Flows, config *config.Flows) (*Flows, error) {
	duration := func(s string, d time.Duration) (time.Duration, error) {
		if s == "" {
			return d, nil
		}
		return time.ParseDuration(s)
	}

	timeout, err := duration(config.Timeout, defaultTimeout)
	if err != nil {
		logp.Err("failed to parse flow timeout: %v", err)
		return nil, err
	}

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
	debugf("lock flows")
	f.table.Lock()
}

func (f *Flows) Unlock() {
	debugf("unlock flows")
	f.table.Unlock()
}

func (f *Flows) Get(id *FlowID) *Flow {
	debugf("get flow")
	if id.flow.stats == nil {
		debugf("lookup flow: %v => %v", id.flowIDMeta, id.flowID)
		id.flow = f.table.get(id, f.counterReg)
	}
	return &id.flow
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

func (f *Flows) NewUint(name string) (*Uint, error) {
	return f.counterReg.newUint(name)
}

func (f *Flows) NewFloat(name string) (*Float, error) {
	return f.counterReg.newFloat(name)
}
