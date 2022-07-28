package beater

import (
	"github.com/elastic/beats/v7/heartbeat/ftestutils"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFactory(t *testing.T) {
	time.Sleep(time.Second * 2)

	mtr, err := runMonitorOnce(t, mapstr.M{
		"id":       "testId",
		"name":     "testName",
		"type":     "http",
		"schedule": "@every 1m",
		"urls":     []string{"https://elastic.github.io/synthetics-demo/"},
	})
	defer mtr.Close()
	require.NoError(t, err)
	require.NotNil(t, mtr.Monitor)
	mtr.Wait()
	require.Len(t, mtr.Events(), 10)
}

type MonitorTestRun struct {
	Monitor *monitors.Monitor
	Events  func() []*beat.Event
	Wait    func()
	Close   func()
}

func runMonitorOnce(t *testing.T, monitorConfig mapstr.M) (mtr *MonitorTestRun, err error) {
	mtr = &MonitorTestRun{}

	// make a pipeline
	pipe := &ftestutils.MockPipelineConnector{}
	// pass it to the factory
	f, sched, closeFactory := makeTestFactory()
	conf, err := config.NewConfigFrom(monitorConfig)
	require.NoError(t, err)

	mIface, err := f.Create(pipe, conf)
	mtr.Monitor, _ = mIface.(*monitors.Monitor)

	// start the monitor
	mtr.Monitor.Start()
	// wait for the monitor to stop
	// wait for the pipeline to clear (ack)
	mtr.Wait = func() {
		sched.WaitForRunOnce()
		mtr.Monitor.Stop()
		closeFactory()
	}
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

	return monitors.NewFactory(info, sched.Add, plugin.GlobalPluginsReg, true),
		sched,
		sched.Stop
}
