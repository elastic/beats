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
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/hbtest"

	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/watcher"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func Test_checkMonitorConfig(t *testing.T) {
	type args struct {
		config    *common.Config
		registrar *pluginsReg
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkMonitorConfig(tt.args.config, tt.args.registrar); (err != nil) != tt.wantErr {
				t.Errorf("checkMonitorConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type MockBeatClient struct {
	publishes []beat.Event
	closed    bool
	mtx       sync.Mutex
}

func (c *MockBeatClient) Publish(e beat.Event) {
	c.PublishAll([]beat.Event{e})
}

func (c *MockBeatClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	for _, e := range events {
		c.publishes = append(c.publishes, e)
	}
}

func (c *MockBeatClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
	return nil
}

type MockPipelineConnector struct {
	clients []*MockBeatClient
	mtx     sync.Mutex
}

func (pc *MockPipelineConnector) Connect() (beat.Client, error) {
	return pc.ConnectWith(beat.ClientConfig{})
}

func (pc *MockPipelineConnector) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	c := &MockBeatClient{}

	pc.clients = append(pc.clients, c)

	return c, nil
}

type MockJob struct{}

func (mj *MockJob) Name() string {
	return "mock"
}

func (mj *MockJob) Run() (beat.Event, []jobRunner, error) {
	return MakeSimpleJob(JobSettings{}, func() (common.MapStr, error) {
		return common.MapStr{
			"foo": "bar",
		}, nil
	}).Run()
}

func createMockJob(name string, cfg *common.Config) ([]Job, error) {
	return []Job{&MockJob{}}, nil
}

func simpleHTTPConf(t *testing.T, url string) *common.Config {
	conf, err := common.NewConfigFrom(map[string]interface{}{
		"type": "http",
		"urls": []string{url},
	})
	require.NoError(t, err)

	return conf
}

func Test_newMonitor(t *testing.T) {
	server := httptest.NewServer(hbtest.HelloWorldHandler(200))

	serverMonConf := simpleHTTPConf(t, server.URL)

	type args struct {
		config            *common.Config
		registrar         *pluginsReg
		pipelineConnector beat.PipelineConnector
		scheduler         *scheduler.Scheduler
	}
	tests := []struct {
		name    string
		args    args
		want    *Monitor
		wantErr bool
	}{
		{
			"simple",
			args{
				serverMonConf,
				globalPluginsReg,
				&MockPipelineConnector{},
				&scheduler.Scheduler{},
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newMonitor(tt.args.config, tt.args.registrar, tt.args.pipelineConnector, tt.args.scheduler)
			if (err != nil) != tt.wantErr {
				t.Errorf("newMonitor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newMonitor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMonitor_makeTasks(t *testing.T) {
	type fields struct {
		name              string
		config            *common.Config
		registrar         *pluginsReg
		uniqueName        string
		scheduler         *scheduler.Scheduler
		jobTasks          []*task
		enabled           bool
		internalsMtx      sync.Mutex
		watchPollTasks    []*task
		watch             watcher.Watch
		pipelineConnector beat.PipelineConnector
	}
	type args struct {
		config *common.Config
		jobs   []Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*task
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{
				name:              tt.fields.name,
				config:            tt.fields.config,
				registrar:         tt.fields.registrar,
				uniqueName:        tt.fields.uniqueName,
				scheduler:         tt.fields.scheduler,
				jobTasks:          tt.fields.jobTasks,
				enabled:           tt.fields.enabled,
				internalsMtx:      tt.fields.internalsMtx,
				watchPollTasks:    tt.fields.watchPollTasks,
				watch:             tt.fields.watch,
				pipelineConnector: tt.fields.pipelineConnector,
			}
			got, err := m.makeTasks(tt.args.config, tt.args.jobs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Monitor.makeTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Monitor.makeTasks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMonitor_makeWatchTasks(t *testing.T) {
	type fields struct {
		name              string
		config            *common.Config
		registrar         *pluginsReg
		uniqueName        string
		scheduler         *scheduler.Scheduler
		jobTasks          []*task
		enabled           bool
		internalsMtx      sync.Mutex
		watchPollTasks    []*task
		watch             watcher.Watch
		pipelineConnector beat.PipelineConnector
	}
	type args struct {
		monitorPlugin pluginBuilder
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{
				name:              tt.fields.name,
				config:            tt.fields.config,
				registrar:         tt.fields.registrar,
				uniqueName:        tt.fields.uniqueName,
				scheduler:         tt.fields.scheduler,
				jobTasks:          tt.fields.jobTasks,
				enabled:           tt.fields.enabled,
				internalsMtx:      tt.fields.internalsMtx,
				watchPollTasks:    tt.fields.watchPollTasks,
				watch:             tt.fields.watch,
				pipelineConnector: tt.fields.pipelineConnector,
			}
			if err := m.makeWatchTasks(tt.args.monitorPlugin); (err != nil) != tt.wantErr {
				t.Errorf("Monitor.makeWatchTasks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMonitor_Start(t *testing.T) {
	type fields struct {
		name              string
		config            *common.Config
		registrar         *pluginsReg
		uniqueName        string
		scheduler         *scheduler.Scheduler
		jobTasks          []*task
		enabled           bool
		internalsMtx      sync.Mutex
		watchPollTasks    []*task
		watch             watcher.Watch
		pipelineConnector beat.PipelineConnector
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{
				name:              tt.fields.name,
				config:            tt.fields.config,
				registrar:         tt.fields.registrar,
				uniqueName:        tt.fields.uniqueName,
				scheduler:         tt.fields.scheduler,
				jobTasks:          tt.fields.jobTasks,
				enabled:           tt.fields.enabled,
				internalsMtx:      tt.fields.internalsMtx,
				watchPollTasks:    tt.fields.watchPollTasks,
				watch:             tt.fields.watch,
				pipelineConnector: tt.fields.pipelineConnector,
			}
			m.Start()
		})
	}
}

func TestMonitor_Stop(t *testing.T) {
	type fields struct {
		name              string
		config            *common.Config
		registrar         *pluginsReg
		uniqueName        string
		scheduler         *scheduler.Scheduler
		jobTasks          []*task
		enabled           bool
		internalsMtx      sync.Mutex
		watchPollTasks    []*task
		watch             watcher.Watch
		pipelineConnector beat.PipelineConnector
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{
				name:              tt.fields.name,
				config:            tt.fields.config,
				registrar:         tt.fields.registrar,
				uniqueName:        tt.fields.uniqueName,
				scheduler:         tt.fields.scheduler,
				jobTasks:          tt.fields.jobTasks,
				enabled:           tt.fields.enabled,
				internalsMtx:      tt.fields.internalsMtx,
				watchPollTasks:    tt.fields.watchPollTasks,
				watch:             tt.fields.watch,
				pipelineConnector: tt.fields.pipelineConnector,
			}
			m.Stop()
		})
	}
}
