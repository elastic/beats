// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package monitors

import (
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/validator"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
)

func makeMockFactory(pluginsReg *plugin.PluginsReg) (factory *RunnerFactory, sched *scheduler.Scheduler, close func()) {
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
	return NewFactory(FactoryParams{
			BeatInfo:    info,
			AddTask:     sched.Add,
			StateLoader: monitorstate.NilStateLoader,
			PluginsReg:  pluginsReg,
			PipelineClientFactory: func(pipeline beat.Pipeline) (pipeline.ISyncClient, error) {
				c, _ := pipeline.Connect()
				return SyncPipelineClientAdaptor{C: c}, nil
			},
		}),
		sched,
		sched.Stop
}

type mockClient struct {
	publishLog   []*beat.Event
	pipeline     beat.Pipeline
	closed       bool
	mtx          sync.Mutex
	clientConfig beat.ClientConfig
}

func (c *mockClient) IsClosed() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.closed
}

func (c *mockClient) Publish(e beat.Event) {
	if c.clientConfig.Processing.Processor != nil {
		outE, _ := c.clientConfig.Processing.Processor.Run(&e)
		e = *outE
	}
	c.PublishAll([]beat.Event{e})
}

func (c *mockClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for _, e := range events {
		eLocal := e
		c.publishLog = append(c.publishLog, &eLocal)
	}
}

func (c *mockClient) Wait() {
}

func (c *mockClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
	return nil
}

func (c *mockClient) PublishedEvents() []*beat.Event {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.publishLog
}

type MockPipeline struct {
	Clients []*mockClient
	mtx     sync.Mutex
}

func (pc *MockPipeline) Connect() (beat.Client, error) {
	return pc.ConnectWith(beat.ClientConfig{})
}

func (pc *MockPipeline) ConnectWith(cc beat.ClientConfig) (beat.Client, error) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	c := &mockClient{pipeline: pc, clientConfig: cc}

	pc.Clients = append(pc.Clients, c)

	return c, nil
}

// Convenience function for tests
func (pc *MockPipeline) ConnectSync() pipeline.ISyncClient {
	c, _ := pc.Connect()
	return SyncPipelineClientAdaptor{C: c}
}

func (pc *MockPipeline) PublishedEvents() []*beat.Event {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	var events []*beat.Event
	for _, c := range pc.Clients {
		events = append(events, c.PublishedEvents()...)
	}

	return events
}

func baseMockEventMonitorValidator(id string, name string, status string) validator.Validator {
	var idMatcher isdef.IsDef
	if id == "" {
		idMatcher = isdef.IsStringMatching(regexp.MustCompile(`^auto-test-.*`))
	} else {
		idMatcher = isdef.IsEqual(id)
	}
	return lookslike.MustCompile(map[string]interface{}{
		"monitor": map[string]interface{}{
			"id":          idMatcher,
			"name":        name,
			"type":        "test",
			"duration.us": hbtestllext.IsInt64,
			"status":      status,
			"check_group": isdef.IsString,
		},
	})
}

func mockEventMonitorValidator(id string, name string) validator.Validator {
	return lookslike.Strict(lookslike.Compose(
		baseMockEventMonitorValidator(id, name, "up"),
		hbtestllext.MonitorTimespanValidator,
		hbtest.SummaryChecks(1, 0),
		lookslike.MustCompile(mockEventCustomFields()),
	))
}

func mockEventCustomFields() map[string]interface{} {
	return mapstr.M{"foo": "bar"}
}

func createMockJob() []jobs.Job {
	j := jobs.MakeSimpleJob(func(event *beat.Event) error {
		eventext.MergeEventFields(event, mockEventCustomFields())
		return nil
	})

	return []jobs.Job{j}
}

func mockPluginBuilder() (plugin.PluginFactory, *atomic.Int, *atomic.Int) {
	reg := monitoring.NewRegistry()

	built := atomic.NewInt(0)
	closed := atomic.NewInt(0)

	return plugin.PluginFactory{
			Name:    "test",
			Aliases: []string{"testAlias"},
			Make: func(s string, config *config.C) (plugin.Plugin, error) {
				built.Inc()
				// Declare a real config block with a required attr so we can see what happens when it doesn't work
				unpacked := struct {
					URLs []string `config:"urls" validate:"required"`
				}{}

				// track all closes, even on error
				closer := func() error {
					closed.Inc()
					return nil
				}

				err := config.Unpack(&unpacked)
				if err != nil {
					return plugin.Plugin{DoClose: closer}, err
				}
				j := createMockJob()

				return plugin.Plugin{Jobs: j, DoClose: closer, Endpoints: 1}, nil
			},
			Stats: plugin.NewPluginCountersRecorder("test", reg)},
		built,
		closed
}

func mockPluginsReg() (p *plugin.PluginsReg, built *atomic.Int, closed *atomic.Int) {
	reg := plugin.NewPluginsReg()
	builder, built, closed := mockPluginBuilder()
	_ = reg.Add(builder)
	return reg, built, closed
}

func mockPluginConf(t *testing.T, id string, name string, schedule string, url string) *config.C {
	confMap := map[string]interface{}{
		"type":     "test",
		"urls":     []string{url},
		"schedule": schedule,
		"name":     name,
	}

	// Optional to let us simulate this key missing
	if id != "" {
		confMap["id"] = id
	}

	conf, err := config.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

// mockBadPluginConf returns a conf with an invalid plugin config.
// This should fail after the generic plugin checks fail since the HTTP plugin requires 'urls' to be set.
func mockBadPluginConf(t *testing.T, id string) *config.C {
	confMap := map[string]interface{}{
		"type":        "test",
		"notanoption": []string{"foo"},
	}

	if id != "" {
		confMap["id"] = id
	}

	conf, err := config.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

func mockInvalidPluginConf(t *testing.T) *config.C {
	confMap := map[string]interface{}{
		"hoeutnheou": "oueanthoue",
	}

	conf, err := config.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

func mockInvalidPluginConfWithStdFields(t *testing.T, id string, name string, schedule string) *config.C {
	confMap := map[string]interface{}{
		"type":     "test",
		"id":       id,
		"name":     name,
		"schedule": schedule,
	}

	conf, err := config.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}
