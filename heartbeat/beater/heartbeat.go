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
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/heartbeat/config"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

// Heartbeat represents the root datastructure of this beat.
type Heartbeat struct {
	done chan struct{}
	// config is used for iterating over elements of the config.
	config          config.Config
	scheduler       *scheduler.Scheduler
	monitorReloader *cfgfile.Reloader
}

// New creates a new heartbeat.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	cfgwarn.Beta("Heartbeat is beta software")

	parsedConfig := config.DefaultConfig
	if err := rawConfig.Unpack(&parsedConfig); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	limit := parsedConfig.Scheduler.Limit
	locationName := parsedConfig.Scheduler.Location
	if locationName == "" {
		locationName = "Local"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return nil, err
	}

	scheduler := scheduler.NewWithLocation(limit, location)

	bt := &Heartbeat{
		done:      make(chan struct{}),
		config:    parsedConfig,
		scheduler: scheduler,
	}
	return bt, nil
}

// Run executes the beat.
func (bt *Heartbeat) Run(b *beat.Beat) error {
	logp.Info("heartbeat is running! Hit CTRL-C to stop it.")

	err := bt.RunStaticMonitors(b)
	if err != nil {
		return err
	}

	if bt.config.ConfigMonitors.Enabled() {
		bt.monitorReloader = cfgfile.NewReloader(b.Publisher, bt.config.ConfigMonitors)
		defer bt.monitorReloader.Stop()

		err := bt.RunDynamicMonitors(b)
		if err != nil {
			return err
		}
	}

	if err := bt.scheduler.Start(); err != nil {
		return err
	}
	defer bt.scheduler.Stop()

	<-bt.done

	logp.Info("Shutting down.")
	return nil
}

// RunStaticMonitors runs the `heartbeat.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunStaticMonitors(b *beat.Beat) error {
	factory := monitors.NewFactory(bt.scheduler, true)

	for _, cfg := range bt.config.Monitors {
		created, err := factory.Create(b.Publisher, cfg, nil)
		if err != nil {
			return errors.Wrap(err, "could not create monitor")
		}
		created.Start()
	}
	return nil
}

// RunDynamicMonitors runs the `heartbeat.config.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunDynamicMonitors(b *beat.Beat) (err error) {
	factory := monitors.NewFactory(bt.scheduler, false)

	// Check monitor configs
	if err := bt.monitorReloader.Check(factory); err != nil {
		return err
	}

	// Execute the monitor
	go bt.monitorReloader.Run(factory)

	return nil
}

// Stop stops the beat.
func (bt *Heartbeat) Stop() {
	close(bt.done)
}
