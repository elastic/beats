// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type ScenarioRun func() (config mapstr.M, close func(), err error)

type Scenario struct {
	Name   string
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

func (sdb ScenarioDB) Init() {
	sdb.initOnce.Do(func() {
		for _, s := range sdb.All {
			for _, t := range s.Tags {
				sdb.ByTag[t] = append(sdb.ByTag[t], s)
			}
		}
	})
}

func (sdb ScenarioDB) RunAll(t *testing.T, callback func(*MonitorTestRun, error)) {
	sdb.Init()
	for _, s := range sdb.All {
		s.Run(t, callback)
	}
}

func (sdb ScenarioDB) RunTag(t *testing.T, tagName string, callback func(*MonitorTestRun, error)) {
	sdb.Init()
	if len(sdb.ByTag[tagName]) < 1 {
		require.Failf(t, "no scenarios have tags matching %s", tagName)
	}
	for _, s := range sdb.ByTag[tagName] {
		s.Run(t, callback)
	}
}

var Scenarios = ScenarioDB{
	initOnce: &sync.Once{},
	ByTag:    map[string][]Scenario{},
	All: []Scenario{
		{
			Name: "http-simple",
			Tags: []string{"lightweight", "http"},
			Runner: func() (config mapstr.M, close func(), err error) {
				server := httptest.NewServer(hbtest.HelloWorldHandler(200))
				config = mapstr.M{
					"id":       "http-test-id",
					"name":     "http-test-name",
					"type":     "http",
					"schedule": "@every 1m",
					"urls":     []string{server.URL},
				}
				return config, server.Close, nil
			},
		},
		{
			Name: "tcp-simple",
			Tags: []string{"lightweight", "tcp"},
			Runner: func() (config mapstr.M, close func(), err error) {
				server := httptest.NewServer(hbtest.HelloWorldHandler(200))
				parsedUrl, err := url.Parse(server.URL)
				if err != nil {
					panic(fmt.Sprintf("URL %s should always be parsable: %s", server.URL, err))
				}
				config = mapstr.M{
					"id":       "tcp-test-id",
					"name":     "tcp-test-name",
					"type":     "tcp",
					"schedule": "@every 1m",
					"hosts":    []string{fmt.Sprintf("%s:%s", parsedUrl.Host, parsedUrl.Port())},
				}
				return config, server.Close, nil
			},
		},
		{
			Name: "simple-icmp",
			Tags: []string{"icmp"},
			Runner: func() (config mapstr.M, close func(), err error) {
				return mapstr.M{
					"id":       "icmp-test-id",
					"name":     "icmp-test-name",
					"type":     "icmp",
					"schedule": "@every 1m",
					"hosts":    []string{"127.0.0.1"},
				}, func() {}, nil
			},
		},
		{
			Name: "simple-browser",
			Tags: []string{"browser", "browser-inline"},
			Runner: func() (config mapstr.M, close func(), err error) {
				err = os.Setenv("ELASTIC_SYNTHETICS_CAPABLE", "true")
				if err != nil {
					return nil, nil, err
				}
				server := httptest.NewServer(hbtest.HelloWorldHandler(200))
				config = mapstr.M{
					"id":       "browser-test-id",
					"name":     "browser-test-name",
					"type":     "browser",
					"schedule": "@every 1m",
					"hosts":    []string{"127.0.0.1"},
					"source": mapstr.M{
						"inline": mapstr.M{
							"script": fmt.Sprintf("step('load server', async () => {await page.goto('%s')})", server.URL),
						},
					},
				}
				return config, server.Close, nil
			},
		},
	},
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
	f, sched, closeFactory := makeTestFactory()
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
	// wait for the monitor to stop
	// wait for the pipeline to clear (ack)
	mtr.Wait = func() {
		time.Sleep(time.Second)
		sched.WaitForRunOnce()
		mtr.Monitor.Stop()
		closeFactory()
	}
	mtr.Close = closeFactory
	return mtr, err
}

func makeTestFactory() (factory *monitors.RunnerFactory, sched *scheduler.Scheduler, close func()) {
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
