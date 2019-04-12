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
	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
)

// RunnerFactory that can be used to create cfg.Runner cast versions of Monitor
// suitable for config reloading.
type RunnerFactory struct {
	sched        *scheduler.Scheduler
	allowWatches bool
}

// NewFactory takes a scheduler and creates a RunnerFactory that can create cfgfile.Runner(Monitor) objects.
func NewFactory(sched *scheduler.Scheduler, allowWatches bool) *RunnerFactory {
	return &RunnerFactory{sched, allowWatches}
}

// Create makes a new Runner for a new monitor with the given Config.
func (f *RunnerFactory) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	monitor, err := newMonitor(c, globalPluginsReg, p, f.sched, f.allowWatches, meta)
	return monitor, err
}

// CheckConfig checks to see if the given monitor config is valid.
func (f *RunnerFactory) CheckConfig(config *common.Config) error {
	return checkMonitorConfig(config, globalPluginsReg, f.allowWatches)
}
