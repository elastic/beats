// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
)

type ScenarioRun func() (config mapstr.M, close func(), err error)

type Scenario struct {
	Name   string
	Type   string
	Runner ScenarioRun
	Tags   []string
}

func (s Scenario) Run(t *testing.T, callback func(mtr *MonitorTestRun, err error)) {
	cfgMap, rClose, err := s.Runner()
	defer rClose()
	if err != nil {
		callback(nil, err)
		return
	}

	t.Run(s.Name, func(t *testing.T) {
		t.Parallel()
		mtr, err := runMonitorOnce(t, cfgMap)
		mtr.Wait()
		callback(mtr, err)
		mtr.Close()
	})

}

type ScenarioDB struct {
	All      []Scenario
	ByTag    map[string][]Scenario
	initOnce *sync.Once
}

func (sdb *ScenarioDB) Init() {
	var prunedList []Scenario
	browserCapable := os.Getenv("ELASTIC_SYNTHETICS_CAPABLE") == "true"
	icmpCapable := os.Getenv("ELASTIC_ICMP_CAPABLE") == "true"
	sdb.initOnce.Do(func() {
		for _, s := range sdb.All {
			if s.Type == "browser" && !browserCapable {
				continue
			}
			if s.Type == "icmp" && !icmpCapable {
				continue
			}
			prunedList = append(prunedList, s)

			for _, t := range s.Tags {
				sdb.ByTag[t] = append(sdb.ByTag[t], s)
			}
		}
	})
	sdb.All = prunedList
}

func (sdb *ScenarioDB) RunAll(t *testing.T, callback func(*MonitorTestRun, error)) {
	sdb.Init()
	for _, s := range sdb.All {
		s.Run(t, callback)
	}
}

func (sdb *ScenarioDB) RunTag(t *testing.T, tagName string, callback func(*MonitorTestRun, error)) {
	sdb.Init()
	if len(sdb.ByTag[tagName]) < 1 {
		require.Failf(t, "no scenarios have tags matching %s", tagName)
	}
	for _, s := range sdb.ByTag[tagName] {
		s.Run(t, callback)
	}
}

type MonitorTestRun struct {
	StdFields stdfields.StdMonitorFields
	Config    mapstr.M
	Monitor   *monitors.Monitor
	Events    func() []*beat.Event
	Wait      func()
	Close     func()
}

func runMonitorOnce(t *testing.T, monitorConfig mapstr.M) (mtr *MonitorTestRun, err error) {
	mtr = &MonitorTestRun{
		Config:    monitorConfig,
		StdFields: stdfields.StdMonitorFields{},
	}

	// make a pipeline
	pipe := &monitors.MockPipeline{}
	// pass it to the factory
	f, sched, closeFactory := setupFactoryAndSched()
	conf, err := config.NewConfigFrom(monitorConfig)
	require.NoError(t, err)
	err = conf.Unpack(&mtr.StdFields)
	require.NoError(t, err)

	mIface, err := f.Create(pipe, conf)
	require.NoError(t, err)
	mtr.Monitor = mIface.(*monitors.Monitor)
	require.NotNil(t, mtr.Monitor, "could not convert to monitor %v", mIface)
	mtr.Events = pipe.PublishedEvents

	// start the monitor
	mtr.Monitor.Start()
	mtr.Wait = func() {
		// wait for the monitor to stop
		sched.WaitForRunOnce()
		// stop the monitor itself
		mtr.Monitor.Stop()
		closeFactory()
	}
	mtr.Close = closeFactory
	return mtr, err
}

func setupFactoryAndSched() (factory *monitors.RunnerFactory, sched *scheduler.Scheduler, close func()) {
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

	return monitors.NewFactory(info, sched.Add, plugin.GlobalPluginsReg, func(pipeline beat.Pipeline) (pipeline.ISyncClient, error) {
			c, _ := pipeline.Connect()
			return monitors.SyncPipelineClientAdaptor{C: c}, nil
		}),
		sched,
		sched.Stop
}
