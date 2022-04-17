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
	"sync"

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/heartbeat/monitors/jobs"
	"github.com/menderesk/beats/v7/heartbeat/monitors/plugin"
	"github.com/menderesk/beats/v7/heartbeat/monitors/stdfields"
	"github.com/menderesk/beats/v7/heartbeat/monitors/wrappers"
	"github.com/menderesk/beats/v7/heartbeat/scheduler"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/publisher/pipeline"
)

// ErrMonitorDisabled is returned when the monitor plugin is marked as disabled.
var ErrMonitorDisabled = errors.New("monitor not loaded, plugin is disabled")

const (
	MON_INIT = iota
	MON_STARTED
	MON_STOPPED
)

type WrappedClient struct {
	Publish func(event beat.Event)
	Close   func() error
	wait    func()
}

// Monitor represents a configured recurring monitoring configuredJob loaded from a config file. Starting it
// will cause it to run with the given scheduler until Stop() is called.
type Monitor struct {
	stdFields      stdfields.StdMonitorFields
	pluginName     string
	config         *common.Config
	addTask        scheduler.AddTask
	configuredJobs []*configuredJob
	enabled        bool
	state          int
	// endpoints is a count of endpoints this monitor measures.
	endpoints int
	// internalsMtx is used to synchronize access to critical
	// internal datastructures
	internalsMtx sync.Mutex
	close        func() error

	pipelineConnector beat.PipelineConnector

	// stats is the countersRecorder used to record lifecycle events
	// for global metrics + telemetry
	stats plugin.RegistryRecorder

	runOnce bool
}

// String prints a description of the monitor in a threadsafe way. It is important that this use threadsafe
// values because it may be invoked from another thread in cfgfile/runner.
func (m *Monitor) String() string {
	return fmt.Sprintf("Monitor<pluginName: %s, enabled: %t>", m.stdFields.Name, m.enabled)
}

func checkMonitorConfig(config *common.Config, registrar *plugin.PluginsReg) error {
	_, err := newMonitor(config, registrar, nil, nil, nil, false)

	return err
}

// newMonitor creates a new monitor, without leaking resources in the event of an error.
// you do not need to call Stop(), it will be safely garbage collected unless Start is called.
func newMonitor(
	config *common.Config,
	registrar *plugin.PluginsReg,
	pipelineConnector beat.PipelineConnector,
	taskAdder scheduler.AddTask,
	onStop func(*Monitor),
	runOnce bool,
) (*Monitor, error) {
	m, err := newMonitorUnsafe(config, registrar, pipelineConnector, taskAdder, onStop, runOnce)
	if m != nil && err != nil {
		m.Stop()
	}
	return m, err
}

// newMonitorUnsafe is the unsafe way of creating a new monitor because it may return a monitor instance along with an
// error without freeing monitor resources. m.Stop() must always be called on a non-nil monitor to free resources.
func newMonitorUnsafe(
	config *common.Config,
	registrar *plugin.PluginsReg,
	pipelineConnector beat.PipelineConnector,
	addTask scheduler.AddTask,
	onStop func(*Monitor),
	runOnce bool,
) (*Monitor, error) {
	// Extract just the Id, Type, and Enabled fields from the config
	// We'll parse things more precisely later once we know what exact type of
	// monitor we have
	standardFields, err := stdfields.ConfigToStdMonitorFields(config)
	if err != nil {
		return nil, err
	}

	if !config.Enabled() {
		return nil, fmt.Errorf("monitor '%s' with id '%s' skipped: %w", standardFields.Name, standardFields.ID, ErrMonitorDisabled)
	}

	pluginFactory, found := registrar.Get(standardFields.Type)
	if !found {
		return nil, fmt.Errorf("monitor type %v does not exist, valid types are %v", standardFields.Type, registrar.MonitorNames())
	}

	m := &Monitor{
		stdFields:         standardFields,
		pluginName:        pluginFactory.Name,
		addTask:           addTask,
		configuredJobs:    []*configuredJob{},
		pipelineConnector: pipelineConnector,
		internalsMtx:      sync.Mutex{},
		config:            config,
		stats:             pluginFactory.Stats,
		state:             MON_INIT,
		runOnce:           runOnce,
	}

	if m.stdFields.ID == "" {
		// If there's no explicit ID generate one
		hash, err := m.configHash()
		if err != nil {
			return m, err
		}
		m.stdFields.ID = fmt.Sprintf("auto-%s-%#X", m.stdFields.Type, hash)
	}

	p, err := pluginFactory.Create(config)

	m.close = func() error {
		if onStop != nil {
			onStop(m)
		}
		return p.Close()
	}

	// If we've hit an error at this point, still run on schedule, but always return an error.
	// This way the error is clearly communicated through to kibana.
	// Since the error is not recoverable in these instances, the user will need to reconfigure
	// the monitor, which will destroy and recreate it in heartbeat, thus clearing this error.
	//
	// Note: we do this at this point, and no earlier, because at a minimum we need the
	// standard monitor fields (id, name and schedule) to deliver an error to kibana in a way
	// that it can render.
	if err != nil {
		// Note, needed to hoist err to this scope, not just to add a prefix
		fullErr := fmt.Errorf("job could not be initialized: %s", err)
		// A placeholder job that always returns an error
		p.Jobs = []jobs.Job{func(event *beat.Event) ([]jobs.Job, error) {
			return nil, fullErr
		}}
	}

	wrappedJobs := wrappers.WrapCommon(p.Jobs, m.stdFields)
	m.endpoints = p.Endpoints

	m.configuredJobs, err = m.makeTasks(config, wrappedJobs)
	if err != nil {
		return m, err
	}

	return m, nil
}

func (m *Monitor) configHash() (uint64, error) {
	unpacked := map[string]interface{}{}
	err := m.config.Unpack(unpacked)
	if err != nil {
		return 0, err
	}
	hash, err := hashstructure.Hash(unpacked, nil)
	if err != nil {
		return 0, err
	}

	return hash, nil
}

func (m *Monitor) makeTasks(config *common.Config, jobs []jobs.Job) ([]*configuredJob, error) {
	mtConf := jobConfig{}
	if err := config.Unpack(&mtConf); err != nil {
		return nil, errors.Wrap(err, "invalid config, could not unpack monitor config")
	}

	var mTasks []*configuredJob
	for _, job := range jobs {
		t, err := newConfiguredJob(job, mtConf, m)
		if err != nil {
			// Failure to compile monitor processors should not crash hb or prevent progress
			if _, ok := err.(ProcessorsError); ok {
				logp.Critical("Failed to load monitor processors: %v", err)
				continue
			}

			return nil, err
		}

		mTasks = append(mTasks, t)
	}

	return mTasks, nil
}

// Start starts the monitor's execution using its configured scheduler.
func (m *Monitor) Start() {
	m.internalsMtx.Lock()
	defer m.internalsMtx.Unlock()

	for _, t := range m.configuredJobs {
		if m.runOnce {
			client, err := pipeline.NewSyncClient(logp.NewLogger("monitor_task"), t.monitor.pipelineConnector, beat.ClientConfig{})
			if err != nil {
				logp.Err("could not start monitor: %v", err)
				continue
			}
			t.Start(&WrappedClient{
				Publish: func(event beat.Event) {
					client.Publish(event)
				},
				Close: client.Close,
				wait:  client.Wait,
			})
		} else {
			client, err := m.pipelineConnector.Connect()
			if err != nil {
				logp.Err("could not start monitor: %v", err)
				continue
			}
			t.Start(&WrappedClient{
				Publish: client.Publish,
				Close:   client.Close,
				wait:    func() {},
			})
		}
	}

	m.stats.StartMonitor(int64(m.endpoints))
	m.state = MON_STARTED
}

// Stop stops the monitor without freeing it in global dedup
// needed by dedup itself to avoid a reentrant lock.
func (m *Monitor) Stop() {
	m.internalsMtx.Lock()
	defer m.internalsMtx.Unlock()

	if m.state == MON_STOPPED {
		return
	}

	for _, t := range m.configuredJobs {
		t.Stop()
	}

	if m.close != nil {
		err := m.close()
		if err != nil {
			logp.Error(fmt.Errorf("error closing monitor %s: %w", m.String(), err))
		}
	}

	m.stats.StopMonitor(int64(m.endpoints))
	m.state = MON_STOPPED
}
