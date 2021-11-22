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

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// ErrMonitorDisabled is returned when the monitor plugin is marked as disabled.
var ErrMonitorDisabled = errors.New("monitor not loaded, plugin is disabled")

// Monitor represents a configured recurring monitoring configuredJob loaded from a config file. Starting it
// will cause it to run with the given scheduler until Stop() is called.
type Monitor struct {
	stdFields      stdfields.StdMonitorFields
	pluginName     string
	config         *common.Config
	registrar      *plugin.PluginsReg
	uniqueName     string
	scheduler      *scheduler.Scheduler
	configuredJobs []*configuredJob
	enabled        bool
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
}

// String prints a description of the monitor in a threadsafe way. It is important that this use threadsafe
// values because it may be invoked from another thread in cfgfile/runner.
func (m *Monitor) String() string {
	return fmt.Sprintf("Monitor<pluginName: %s, enabled: %t>", m.stdFields.Name, m.enabled)
}

func checkMonitorConfig(config *common.Config, registrar *plugin.PluginsReg) error {
	m, err := newMonitor(config, registrar, nil, nil)
	if m != nil {
		m.Stop() // Stop the monitor to free up the ID from uniqueness checks
	}
	return err
}

// uniqueMonitorIDs is used to keep track of explicitly configured monitor IDs and ensure no duplication within a
// given heartbeat instance.
var uniqueMonitorIDs sync.Map

// ErrDuplicateMonitorID is returned when a monitor attempts to start using an ID already in use by another monitor.
type ErrDuplicateMonitorID struct{ ID string }

func (e ErrDuplicateMonitorID) Error() string {
	return fmt.Sprintf("monitor ID %s is configured for multiple monitors! IDs must be unique values.", e.ID)
}

// newMonitor Creates a new monitor, without leaking resources in the event of an error.
func newMonitor(
	config *common.Config,
	registrar *plugin.PluginsReg,
	pipelineConnector beat.PipelineConnector,
	scheduler *scheduler.Scheduler,
) (*Monitor, error) {
	m, err := newMonitorUnsafe(config, registrar, pipelineConnector, scheduler)
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
	scheduler *scheduler.Scheduler,
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
		scheduler:         scheduler,
		configuredJobs:    []*configuredJob{},
		pipelineConnector: pipelineConnector,
		internalsMtx:      sync.Mutex{},
		config:            config,
		stats:             pluginFactory.Stats,
	}

	if m.stdFields.ID != "" {
		// Ensure we don't have duplicate IDs
		if _, loaded := uniqueMonitorIDs.LoadOrStore(m.stdFields.ID, m); loaded {
			return m, ErrDuplicateMonitorID{m.stdFields.ID}
		}
	} else {
		// If there's no explicit ID generate one
		hash, err := m.configHash()
		if err != nil {
			return m, err
		}
		m.stdFields.ID = fmt.Sprintf("auto-%s-%#X", m.stdFields.Type, hash)
	}

	p, err := pluginFactory.Create(config)
	m.close = p.Close
	wrappedJobs := wrappers.WrapCommon(p.Jobs, m.stdFields)
	m.endpoints = p.Endpoints

	if err != nil {
		return m, fmt.Errorf("job err %v", err)
	}

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
		t.Start()
	}

	m.stats.StartMonitor(int64(m.endpoints))
}

// Stop stops the Monitor's execution in its configured scheduler.
// This is safe to call even if the Monitor was never started.
func (m *Monitor) Stop() {
	m.internalsMtx.Lock()
	defer m.internalsMtx.Unlock()
	defer m.freeID()

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
}

func (m *Monitor) freeID() {
	// Free up the monitor ID for reuse
	uniqueMonitorIDs.Delete(m.stdFields.ID)
}
