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

package service

import (
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"sync"
)

// MonitorRunnerFactory that can be used to create cfg.MonitorRunner cast versions of Monitor
// suitable for config reloading.
type MonitorRunnerFactory struct {
	monitorsById map[string]*common.Config
	lock         sync.Mutex
	Update       chan bool
}

// NewRunnerFactory takes a scheduler and creates a MonitorRunnerFactory that can create cfgfile.Runner(Monitor) objects.
func NewRunnerFactory() *MonitorRunnerFactory {
	return &MonitorRunnerFactory{
		monitorsById: map[string]*common.Config{},
		lock:         sync.Mutex{},
		Update: make(chan bool),
	}
}

// Create makes a new MonitorRunner for a new monitor with the given Config.
func (f *MonitorRunnerFactory) Create(p beat.Pipeline, c *common.Config) (cfgfile.Runner, error) {
	monFields, err := stdfields.ConfigToStdMonitorFields(c)
	if err != nil {
		return nil, fmt.Errorf("service factory cannot retrieve monitor field from config: %w", err)
	}

	runner := MonitorRunner{
		config:        c,
		runnerFactory: f,
		monitorId:     monFields.ID,
	}
	return runner, nil

}

// CheckConfig checks to see if the given monitor config is valid.
func (f *MonitorRunnerFactory) CheckConfig(config *common.Config) error {
	_, err := stdfields.ConfigToStdMonitorFields(config)
	return err
}

func (f *MonitorRunnerFactory) GetMonitorsById() map[string]*common.Config {
	f.lock.Lock()
	defer f.lock.Unlock()
	monitorsByIdCopy := map[string]*common.Config{}

	for _, monitor := range f.monitorsById{
		monFields, err := stdfields.ConfigToStdMonitorFields(monitor)
		if err != nil {
			logp.Warn("service factory cannot retrieve monitor field from config: %s", err)
		}
		monitorsByIdCopy[monFields.ID] = monitor
	}

	return monitorsByIdCopy
}

type MonitorRunner struct {
	runnerFactory *MonitorRunnerFactory
	monitorId     string
	config        *common.Config
}

func (r MonitorRunner) String() string {
	return fmt.Sprintf("service monitor runner (id: %s )", r.monitorId)
}

func (r MonitorRunner) Start() {
	logp.Info("Monitor service factory has parsed the monitor: %s", r.monitorId)
	r.runnerFactory.lock.Lock()
	defer r.runnerFactory.lock.Unlock()

	r.runnerFactory.monitorsById[r.monitorId] = r.config
	r.runnerFactory.Update <- true
}

func (r MonitorRunner) Stop() {
	logp.Info("Monitor service factory has deleted the monitor: %s", r.monitorId)

	r.runnerFactory.lock.Lock()
	defer r.runnerFactory.lock.Unlock()

	delete(r.runnerFactory.monitorsById, r.monitorId)
	r.runnerFactory.Update <- true
}
