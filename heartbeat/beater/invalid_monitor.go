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

package beater

import (
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// invalidMonitorFactory replaces a monitor whose config cannot be loaded with an
// invalidMonitor. It is used in receiver mode, where one receiver can run every
// monitor of a policy: an invalid monitor must not stop the others, and its
// failure must be reported for as long as its config exists.
type invalidMonitorFactory struct {
	cfgfile.RunnerFactory
	logger *logp.Logger
}

// Unwrap returns the wrapped factory, so the runner list can still use its
// config hash function.
func (f invalidMonitorFactory) Unwrap() cfgfile.RunnerFactory {
	return f.RunnerFactory
}

func (f invalidMonitorFactory) Create(p beat.PipelineConnector, cfg *conf.C) (cfgfile.Runner, error) {
	runner, err := f.RunnerFactory.Create(p, cfg)
	if err == nil || errors.Is(err, monitors.ErrMonitorDisabled) {
		return runner, err
	}
	f.logger.Errorf("monitor could not be created, reporting it as failed: %s", err)
	return &invalidMonitor{err: err}, nil
}

// invalidMonitor stands in for a monitor whose config cannot be loaded. It runs
// nothing, reports the error while started, and clears it once stopped, which
// happens when its config is removed or replaced.
type invalidMonitor struct {
	err error

	mtx      sync.Mutex
	reporter status.StatusReporter
}

func (m *invalidMonitor) SetStatusReporter(reporter status.StatusReporter) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.reporter = reporter
}

func (m *invalidMonitor) Start() { m.updateStatus(status.Failed, m.err.Error()) }

func (m *invalidMonitor) Stop() { m.updateStatus(status.Stopped, "") }

func (m *invalidMonitor) String() string { return fmt.Sprintf("invalid monitor: %s", m.err) }

func (m *invalidMonitor) updateStatus(s status.Status, msg string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.reporter != nil {
		m.reporter.UpdateStatus(s, msg)
	}
}
