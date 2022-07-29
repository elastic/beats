package scenarios

import (
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSimpleScenariosBasicFields(t *testing.T) {
	scenarios := []Scenario{SimpleBrowserScenario, SimpleHTTPScenario, SimpleICMPScenario, SimpleICMPScenario}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("basic fields: %s", scenario.Name), func(t *testing.T) {
			scenario := scenario // scope correctly for parallel test
			t.Parallel()

			_, mtr, err := scenario.Run(t)
			defer mtr.Close()
			require.NoError(t, err)
			if err != nil {
				return
			}

			require.GreaterOrEqual(t, len(mtr.Events()), 1)
			lastCg := ""
			for i, e := range mtr.Events() {
				cg, err := e.GetValue("monitor.check_group")
				require.NoError(t, err)
				cgStr := cg.(string)
				if i == 0 {
					lastCg = cgStr
				} else {
					require.Equal(t, lastCg, cgStr)
				}
			}
		})
	}
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
	pipe := &monitors.MockPipeline{}
	// pass it to the factory
	f, sched, closeFactory := makeTestFactory()
	conf, err := config.NewConfigFrom(monitorConfig)
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
