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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/heartbeat/config"
	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/scheduler"
)

type Heartbeat struct {
	done chan struct{}

	scheduler *scheduler.Scheduler
	manager   *monitorManager
}

func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	logp.Warn("Beta: Heartbeat is beta software")

	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	limit := config.Scheduler.Limit
	locationName := config.Scheduler.Location
	if locationName == "" {
		locationName = "Local"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return nil, err
	}

	sched := scheduler.NewWithLocation(limit, location)
	manager, err := newMonitorManager(b.Publisher, sched, monitors.Registry, config.Monitors)
	if err != nil {
		return nil, err
	}

	bt := &Heartbeat{
		done:      make(chan struct{}),
		scheduler: sched,
		manager:   manager,
	}
	return bt, nil
}

func (bt *Heartbeat) Run(b *beat.Beat) error {
	logp.Info("heartbeat is running! Hit CTRL-C to stop it.")

	if err := bt.scheduler.Start(); err != nil {
		return err
	}
	defer bt.scheduler.Stop()

	<-bt.done

	bt.manager.Stop()

	logp.Info("Shutting down.")
	return nil
}

func (bt *Heartbeat) Stop() {
	close(bt.done)
}
