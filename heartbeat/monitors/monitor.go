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

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"

	"github.com/mitchellh/hashstructure"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// ErrMonitorDisabled is returned when the monitor plugin is marked as disabled.
var ErrMonitorDisabled = fmt.Errorf("monitor not loaded, plugin is disabled")

const (
	MON_INIT = iota
	MON_STARTED
	MON_STOPPED
)

// Monitor represents a configured recurring monitoring configuredJob loaded from a config file. Starting it
// will cause it to run with the given scheduler until Stop() is called.
type Monitor struct {
	stdFields      stdfields.StdMonitorFields
	pluginName     string
	config         *conf.C
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

	// pubClient accepts a generic beat.Client. Pipeline synchronicity is implemented
	// at client wrapper-level
	pubClient beat.Client

	// stats is the countersRecorder used to record lifecycle events
	// for global metrics + telemetry
	stats plugin.RegistryRecorder

	monitorStateTracker *monitorstate.Tracker
}

// String prints a description of the monitor in a threadsafe way. It is important that this use threadsafe
// values because it may be invoked from another thread in cfgfile/runner.
func (m *Monitor) String() string {
	return fmt.Sprintf("Monitor<pluginName: %s, enabled: %t>", m.stdFields.Name, m.enabled)
}

func checkMonitorConfig(config *conf.C, registrar *plugin.PluginsReg) error {
	_, err := newMonitor(config, registrar, nil, nil, monitorstate.NilStateLoader, nil)

	return err
}

// newMonitor creates a new monitor, without leaking resources in the event of an error.
// you do not need to call Stop(), it will be safely garbage collected unless Start is called.
func newMonitor(
	config *conf.C,
	registrar *plugin.PluginsReg,
	pubClient beat.Client,
	taskAdder scheduler.AddTask,
	stateLoader monitorstate.StateLoader,
	onStop func(*Monitor),
) (*Monitor, error) {
	m, err := newMonitorUnsafe(config, registrar, pubClient, taskAdder, stateLoader, onStop)
	if m != nil && err != nil {
		m.Stop()
	}
	return m, err
}

// newMonitorUnsafe is the unsafe way of creating a new monitor because it may return a monitor instance along with an
// error without freeing monitor resources. m.Stop() must always be called on a non-nil monitor to free resources.
func newMonitorUnsafe(
	config *conf.C,
	registrar *plugin.PluginsReg,
	pubClient beat.Client,
	addTask scheduler.AddTask,
	stateLoader monitorstate.StateLoader,
	onStop func(*Monitor),
) (*Monitor, error) {
	// Extract just the Id, Type, and Enabled fields from the config
	// We'll parse things more precisely later once we know what exact type of
	// monitor we have
	standardFields, err := stdfields.ConfigToStdMonitorFields(config)
	if err != nil {
		return nil, err
	}

	pluginFactory, found := registrar.Get(standardFields.Type)
	if !found {
		return nil, fmt.Errorf("monitor type %v does not exist, valid types are %v", standardFields.Type, registrar.MonitorNames())
	}

	m := &Monitor{
		stdFields:           standardFields,
		pluginName:          pluginFactory.Name,
		addTask:             addTask,
		configuredJobs:      []*configuredJob{},
		pubClient:           pubClient,
		internalsMtx:        sync.Mutex{},
		config:              config,
		stats:               pluginFactory.Stats,
		state:               MON_INIT,
		monitorStateTracker: monitorstate.NewTracker(stateLoader, false),
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

	var wrappedJobs []jobs.Job
	if err == nil {
		wrappedJobs = wrappers.WrapCommon(p.Jobs, m.stdFields, stateLoader)
	} else {
		// If we've hit an error at this point, still run on schedule, but always return an error.
		// This way the error is clearly communicated through to kibana.
		// Since the error is not recoverable in these instances, the user will need to reconfigure
		// the monitor, which will destroy and recreate it in heartbeat, thus clearing this error.
		//
		// Note: we do this at this point, and no earlier, because at a minimum we need the
		// standard monitor fields (id, name and schedule) to deliver an error to kibana in a way
		// that it can render.

		// Note, needed to hoist err to this scope, not just to add a prefix
		fullErr := fmt.Errorf("job could not be initialized: %w", err)
		// A placeholder job that always returns an error

		logp.L().Error(fullErr)
		p.Jobs = []jobs.Job{func(event *beat.Event) ([]jobs.Job, error) {
			return nil, fullErr
		}}

		// We need to use the lightweight wrapping for error jobs
		// since browser wrapping won't write summaries, but the fake job here is
		// effectively a lightweight job
		m.stdFields.BadConfig = true
		// No need to retry bad configs
		m.stdFields.MaxAttempts = 1
		wrappedJobs = wrappers.WrapCommon(p.Jobs, m.stdFields, stateLoader)
	}

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

func (m *Monitor) makeTasks(config *conf.C, jobs []jobs.Job) ([]*configuredJob, error) {
	mtConf := jobConfig{}
	if err := config.Unpack(&mtConf); err != nil {
		return nil, fmt.Errorf("invalid config, could not unpack monitor config: %w", err)
	}

	var mTasks = make([]*configuredJob, 0, len(jobs))
	for _, job := range jobs {
		t := newConfiguredJob(job, mtConf, m)
		mTasks = append(mTasks, t)
	}

	return mTasks, nil
}

// Start starts the monitor's execution using its configured scheduler.
func (m *Monitor) Start() {
	m.internalsMtx.Lock()
	defer m.internalsMtx.Unlock()

	for _, t := range m.configuredJobs {
		t.Start(m.pubClient)
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
			logp.L().Errorf("error closing monitor %s: %w", m.String(), err)
		}
	}

	m.stats.StopMonitor(int64(m.endpoints))
	m.state = MON_STOPPED
}
