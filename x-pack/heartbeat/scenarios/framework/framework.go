// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package framework

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"

	hbconfig "github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
)

type ScenarioRun func(t *testing.T) (config mapstr.M, meta ScenarioRunMeta, close func(), err error)
type ScenarioRunMeta struct {
	URL    *url.URL
	Status monitorstate.StateStatus
}

type Scenario struct {
	Name         string
	Type         string
	Runner       ScenarioRun
	Tags         []string
	RunFrom      *hbconfig.LocationWithID
	NumberOfRuns int
	URL          string
}

type Twist struct {
	Name string
	Fn   func(Scenario) Scenario
}

func MakeTwist(name string, fn func(Scenario) Scenario) *Twist {
	return &Twist{
		Name: name,
		Fn: func(s Scenario) Scenario {
			newS := s.clone()
			newS.Name = fmt.Sprintf("%s~<%s>", s.Name, name)
			return fn(newS)
		},
	}
}

func MultiTwist(twists ...*Twist) *Twist {
	return MakeTwist(
		"<~MULTI-TWIST~[",
		func(s Scenario) Scenario {
			res := s
			for _, twist := range twists {
				res = twist.Fn(res)
			}
			return res
		})
}

func (s Scenario) clone() Scenario {
	copy := s
	if s.RunFrom != nil {
		locationCopy := *s.RunFrom
		copy.RunFrom = &locationCopy
	}
	return copy
}

func (s Scenario) Run(t *testing.T, twist *Twist, callback func(t *testing.T, mtr *MonitorTestRun, err error)) {
	runS := s
	if twist != nil {
		runS = twist.Fn(s.clone())
	}

	cfgMap, meta, rClose, err := runS.Runner(t)
	if rClose != nil {
		defer rClose()
	}
	if err != nil {
		callback(t, nil, err)
		return
	}

	t.Run(runS.Name, func(t *testing.T) {
		t.Parallel()

		numberRuns := runS.NumberOfRuns
		if numberRuns < 1 {
			numberRuns = 1 // default to one run
		}

		loaderDB := newLoaderDB()

		var events []*beat.Event

		var err error
		var sf stdfields.StdMonitorFields
		var conf mapstr.M
		for i := 0; i < numberRuns; i++ {
			var mtr *MonitorTestRun
			mtr, err = runMonitorOnce(t, cfgMap, meta, runS.RunFrom, loaderDB.StateLoader())

			mtr.wait()
			events = append(events, mtr.Events()...)

			sf = mtr.StdFields
			conf = mtr.Config

			if lse := LastState(events).State; lse != nil {
				loaderDB.AddState(mtr.StdFields, lse)
			}

			mtr.close()
		}

		sumMtr := MonitorTestRun{
			StdFields: sf,
			Config:    conf,
			Meta:      meta,
			Events: func() []*beat.Event {
				return events
			},
		}

		callback(t, &sumMtr, err)
	})
}

type ScenarioDB struct {
	All      []Scenario
	ByTag    map[string][]Scenario
	initOnce *sync.Once
}

func NewScenarioDB() *ScenarioDB {
	return &ScenarioDB{
		initOnce: &sync.Once{},
		ByTag:    map[string][]Scenario{},
		All:      []Scenario{},
	}

}

func (sdb *ScenarioDB) Init() {
	sdb.initOnce.Do(func() {
		var prunedList []Scenario
		icmpCapable := os.Getenv("ELASTIC_ICMP_CAPABLE") == "true"
		for _, s := range sdb.All {
			if s.Type == "icmp" && !icmpCapable {
				continue
			}
			prunedList = append(prunedList, s)

			for _, t := range s.Tags {
				sdb.ByTag[t] = append(sdb.ByTag[t], s)
			}
		}
		sdb.All = prunedList
	})
}

func (sdb *ScenarioDB) Add(s ...Scenario) {
	sdb.All = append(sdb.All, s...)
}

func (sdb *ScenarioDB) RunAll(t *testing.T, callback func(*testing.T, *MonitorTestRun, error)) {
	sdb.RunAllWithATwist(t, nil, callback)
}

// RunAllWithSeparateTwists runs a list of twists separately, but not chained together.
// This is helpful for building up a test matrix by composing twists.
func (sdb *ScenarioDB) RunAllWithSeparateTwists(t *testing.T, twists []*Twist, callback func(*testing.T, *MonitorTestRun, error)) {
	twists = append(twists, nil) // we also run once with no twists
	for _, twist := range twists {
		sdb.RunAllWithATwist(t, twist, callback)
	}
}

func (sdb *ScenarioDB) RunAllWithATwist(t *testing.T, twist *Twist, callback func(*testing.T, *MonitorTestRun, error)) {
	sdb.Init()
	for _, s := range sdb.All {
		s.Run(t, twist, callback)
	}
}

func (sdb *ScenarioDB) RunTag(t *testing.T, tagName string, callback func(*testing.T, *MonitorTestRun, error)) {
	sdb.RunTagWithATwist(t, tagName, nil, callback)
}

func (sdb *ScenarioDB) RunTagWithATwist(t *testing.T, tagName string, twist *Twist, callback func(*testing.T, *MonitorTestRun, error)) {
	sdb.Init()
	if len(sdb.ByTag[tagName]) < 1 {
		require.Failf(t, "no scenarios have tags matching %s", tagName)
	}
	for _, s := range sdb.ByTag[tagName] {
		s.Run(t, twist, callback)
	}
}

func (sdb *ScenarioDB) RunTagWithSeparateTwists(t *testing.T, tagName string, twists []*Twist, callback func(*testing.T, *MonitorTestRun, error)) {
	for _, twist := range twists {
		sdb.RunTagWithATwist(t, tagName, twist, callback)
	}
}

type MonitorTestRun struct {
	StdFields stdfields.StdMonitorFields
	Meta      ScenarioRunMeta
	Config    mapstr.M
	Events    func() []*beat.Event
	monitor   *monitors.Monitor
	wait      func()
	close     func()
}

func runMonitorOnce(t *testing.T, monitorConfig mapstr.M, meta ScenarioRunMeta, location *hbconfig.LocationWithID, stateLoader monitorstate.StateLoader) (mtr *MonitorTestRun, err error) {
	mtr = &MonitorTestRun{
		Config: monitorConfig,
		Meta:   meta,
		StdFields: stdfields.StdMonitorFields{
			RunFrom: location,
		},
	}

	// make a pipeline
	pipe := &monitors.MockPipeline{}
	// pass it to the factory
	f, sched, closeFactory := setupFactoryAndSched(location, stateLoader)
	conf, err := config.NewConfigFrom(monitorConfig)
	require.NoError(t, err)
	err = conf.Unpack(&mtr.StdFields)
	require.NoError(t, err)

	mIface, err := f.Create(pipe, conf)
	require.NoError(t, err)
	mtr.monitor = mIface.(*monitors.Monitor)
	require.NotNil(t, mtr.monitor, "could not convert to monitor %v", mIface)
	mtr.Events = pipe.PublishedEvents

	// start the monitor
	mtr.monitor.Start()
	mtr.wait = func() {
		// wait for the monitor to stop
		sched.WaitForRunOnce()
		// stop the monitor itself
		mtr.monitor.Stop()
		closeFactory()
	}
	mtr.close = closeFactory
	return mtr, err
}

func setupFactoryAndSched(location *hbconfig.LocationWithID, stateLoader monitorstate.StateLoader) (factory *monitors.RunnerFactory, sched *scheduler.Scheduler, close func()) {
	id, _ := uuid.NewV4()
	eid, _ := uuid.NewV4()
	info := beat.Info{
		Beat:            "heartbeat",
		IndexPrefix:     "heartbeat",
		Version:         beatversion.GetDefaultVersion(),
		ElasticLicensed: true,
		Name:            "heartbeat",
		Hostname:        "localhost",
		ID:              id,
		EphemeralID:     eid,
		FirstStart:      time.Now(),
		StartTime:       time.Now(),
		Monitoring: struct {
			DefaultUsername string
		}{
			DefaultUsername: "test",
		},
	}

	sched = scheduler.Create(
		1,
		monitoring.NewRegistry(),
		time.Local,
		nil,
		true,
	)

	return monitors.NewFactory(monitors.FactoryParams{
			BeatInfo:    info,
			AddTask:     sched.Add,
			StateLoader: stateLoader,
			PluginsReg:  plugin.GlobalPluginsReg,
			PipelineClientFactory: func(pipeline beat.Pipeline) (beat.Client, error) {
				return pipeline.Connect()
			},
			BeatRunFrom: location,
		}),
		sched,
		sched.Stop
}

type stateEvent struct {
	Event *beat.Event
	State *monitorstate.State
}

func AllStates(events []*beat.Event) (stateEvents []stateEvent) {
	for _, e := range events {
		if stateIface, _ := e.Fields.GetValue("state"); stateIface != nil {
			state, ok := stateIface.(*monitorstate.State)
			if !ok {
				panic(fmt.Sprintf("state is not a monitorstate.State, got %v", state))
			}

			se := stateEvent{Event: e, State: state}
			stateEvents = append(stateEvents, se)
		}
	}
	return stateEvents
}

func LastState(events []*beat.Event) *stateEvent {
	all := AllStates(events)

	if len(all) == 0 {
		return nil
	}

	return &all[len(all)-1]
}
