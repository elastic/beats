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
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/watcher"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Monitor represents a configured recurring monitoring task loaded from a config file. Starting it
// will cause it to run with the given scheduler until Stop() is called.
type Monitor struct {
	name       string
	config     *common.Config
	registrar  *pluginsReg
	uniqueName string
	scheduler  *scheduler.Scheduler
	jobTasks   []*task
	enabled    bool
	// internalsMtx is used to synchronize access to critical
	// internal datastructures
	internalsMtx sync.Mutex

	// Watch related fields
	watchPollTasks []*task
	watch          watcher.Watch

	pipeline beat.Pipeline
}

// String prints a description of the monitor in a threadsafe way. It is important that this use threadsafe
// values because it may be invoked from another thread in cfgfile/runner.
func (m *Monitor) String() string {
	return fmt.Sprintf("Monitor<name: %s, enabled: %t>", m.name, m.enabled)
}

func checkMonitorConfig(config *common.Config, registrar *pluginsReg, allowWatches bool) error {
	_, err := newMonitor(config, registrar, nil, nil, allowWatches)
	return err
}

// ErrWatchesDisabled is returned when the user attempts to declare a watch poll file in a
var ErrWatchesDisabled = errors.New("watch poll files are only allowed in heartbeat.yml, not dynamic configs")

func newMonitor(
	config *common.Config,
	registrar *pluginsReg,
	pipeline beat.Pipeline,
	scheduler *scheduler.Scheduler,
	allowWatches bool,
) (*Monitor, error) {
	// Extract just the Type and Enabled fields from the config
	// We'll parse things more precisely later once we know what exact type of
	// monitor we have
	mpi, err := pluginInfo(config)
	if err != nil {
		return nil, err
	}

	monitorPlugin, found := registrar.get(mpi.Type)
	if !found {
		return nil, fmt.Errorf("monitor type %v does not exist", mpi.Type)
	}

	m := &Monitor{
		name:           monitorPlugin.name,
		scheduler:      scheduler,
		jobTasks:       []*task{},
		pipeline:       pipeline,
		watchPollTasks: []*task{},
		internalsMtx:   sync.Mutex{},
		config:         config,
	}

	jobs, err := monitorPlugin.create(config)
	if err != nil {
		return nil, fmt.Errorf("job err %v", err)
	}

	m.jobTasks, err = m.makeTasks(config, jobs)
	if err != nil {
		return nil, err
	}

	err = m.makeWatchTasks(monitorPlugin)
	if err != nil {
		return nil, err
	}

	if len(m.watchPollTasks) > 0 {
		if !allowWatches {
			return nil, ErrWatchesDisabled
		}

		logp.Info(`Obsolete option 'watch.poll_file' declared. This will be removed in a future release. 
See https://www.elastic.co/guide/en/beats/heartbeat/current/configuration-heartbeat-options.html for more info`)
	}

	return m, nil
}

func (m *Monitor) makeTasks(config *common.Config, jobs []Job) ([]*task, error) {
	mtConf := taskConfig{}
	if err := config.Unpack(&mtConf); err != nil {
		return nil, errors.Wrap(err, "invalid config, could not unpack monitor config")
	}

	var mTasks []*task
	for _, job := range jobs {
		t, err := newTask(job, mtConf, m)
		if err != nil {
			// Failure to compile monitor processors should not crash hb or prevent progress
			if _, ok := err.(InvalidMonitorProcessorsError); ok {
				logp.Critical("Failed to load monitor processors: %v", err)
				continue
			}

			return nil, err
		}

		mTasks = append(mTasks, t)
	}

	return mTasks, nil
}

func (m *Monitor) makeWatchTasks(monitorPlugin pluginBuilder) error {
	watchCfg := watcher.DefaultWatchConfig
	err := m.config.Unpack(&watchCfg)
	if err != nil {
		return err
	}

	if len(watchCfg.Path) > 0 {
		m.watch, err = watcher.NewFilePoller(watchCfg.Path, watchCfg.Poll, func(content []byte) {
			var newTasks []*task

			dec := json.NewDecoder(bytes.NewBuffer(content))
			for dec.More() {
				var obj map[string]interface{}
				err = dec.Decode(&obj)
				if err != nil {
					logp.Err("Failed parsing JSON object: %v", err)
					return
				}

				cfg, err := common.NewConfigFrom(obj)
				if err != nil {
					logp.Err("Failed normalizing JSON input: %v", err)
					return
				}

				merged, err := common.MergeConfigs(m.config, cfg)
				if err != nil {
					logp.Err("Could not merge config: %v", err)
					return
				}

				watchJobs, err := monitorPlugin.create(merged)
				if err != nil {
					logp.Err("Could not create job from watch file: %v", err)
				}

				watchTasks, err := m.makeTasks(merged, watchJobs)
				if err != nil {
					logp.Err("Could not make task for config: %v", err)
					return
				}

				newTasks = append(newTasks, watchTasks...)
			}

			m.internalsMtx.Lock()
			defer m.internalsMtx.Unlock()

			for _, t := range m.watchPollTasks {
				t.Stop()
			}
			m.watchPollTasks = newTasks
			for _, t := range m.watchPollTasks {
				t.Start()
			}
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Monitor) Start() {
	m.internalsMtx.Lock()
	defer m.internalsMtx.Unlock()

	for _, t := range m.jobTasks {
		t.Start()
	}

	for _, t := range m.watchPollTasks {
		t.Start()
	}
}

func (m *Monitor) Stop() {
	m.internalsMtx.Lock()
	defer m.internalsMtx.Unlock()

	for _, t := range m.jobTasks {
		t.Stop()
	}

	for _, t := range m.watchPollTasks {
		t.Stop()
	}
}
