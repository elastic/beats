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
	"syscall"
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/hbregistry"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"

	_ "github.com/elastic/beats/v7/heartbeat/security"
)

// Heartbeat represents the root datastructure of this beat.
type Heartbeat struct {
	done chan struct{}
	// config is used for iterating over elements of the config.
	config          config.Config
	scheduler       *scheduler.Scheduler
	monitorReloader *cfgfile.Reloader
	dynamicFactory  *monitors.RunnerFactory
	autodiscover    *autodiscover.Autodiscover
}

// New creates a new heartbeat.
func New(b *beat.Beat, rawConfig *conf.C) (beat.Beater, error) {
	parsedConfig := config.DefaultConfig
	if err := rawConfig.Unpack(&parsedConfig); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
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
	jobConfig := parsedConfig.Jobs

	sched := scheduler.Create(limit, hbregistry.SchedulerRegistry, location, jobConfig, parsedConfig.RunOnce)

	pipelineClientFactory := func(p beat.Pipeline) (pipeline.ISyncClient, error) {
		if parsedConfig.RunOnce {
			client, err := pipeline.NewSyncClient(logp.L(), p, beat.ClientConfig{})
			if err != nil {
				return nil, fmt.Errorf("could not create pipeline sync client for run_once: %w", err)
			}
			return client, nil
		} else {
			client, err := p.Connect()
			return monitors.SyncPipelineClientAdaptor{C: client}, err
		}
	}

	bt := &Heartbeat{
		done:      make(chan struct{}),
		config:    parsedConfig,
		scheduler: sched,
		// dynamicFactory is the factory used for dynamic configs, e.g. autodiscover / reload
		dynamicFactory: monitors.NewFactory(b.Info, sched.Add, plugin.GlobalPluginsReg, pipelineClientFactory),
	}
	return bt, nil
}

// Run executes the beat.
func (bt *Heartbeat) Run(b *beat.Beat) error {
	logp.L().Info("heartbeat is running! Hit CTRL-C to stop it.")
	groups, _ := syscall.Getgroups()
	logp.L().Info("Effective user/group ids: %d/%d, with groups: %v", syscall.Geteuid(), syscall.Getegid(), groups)

	// It is important this appear before we check for run once mode
	// In run once mode we depend on these monitors being loaded, but not other more
	// dynamic types.
	stopStaticMonitors, err := bt.RunStaticMonitors(b)
	if err != nil {
		return err
	}
	defer stopStaticMonitors()

	if bt.config.RunOnce {
		bt.scheduler.WaitForRunOnce()
		logp.L().Info("Ending run_once run")
		return nil
	}

	if b.Manager.Enabled() {
		bt.RunCentralMgmtMonitors(b)
	}

	if bt.config.ConfigMonitors.Enabled() {
		bt.monitorReloader = cfgfile.NewReloader(b.Publisher, bt.config.ConfigMonitors)
		defer bt.monitorReloader.Stop()

		err := bt.RunReloadableMonitors()
		if err != nil {
			return err
		}
	}
	// Configure the beats Manager to start after all the reloadable hooks are initialized
	// and shutdown when the function return.
	if err := b.Manager.Start(); err != nil {
		return err
	}
	defer b.Manager.Stop()

	if bt.config.Autodiscover != nil {
		bt.autodiscover, err = bt.makeAutodiscover(b)
		if err != nil {
			return err
		}

		bt.autodiscover.Start()
		defer bt.autodiscover.Stop()
	}

	defer bt.scheduler.Stop()

	<-bt.done

	logp.L().Info("Shutting down.")
	return nil
}

// RunStaticMonitors runs the `heartbeat.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunStaticMonitors(b *beat.Beat) (stop func(), err error) {
	runners := make([]cfgfile.Runner, 0, len(bt.config.Monitors))
	for _, cfg := range bt.config.Monitors {
		created, err := bt.dynamicFactory.Create(b.Publisher, cfg)
		if err != nil {
			if errors.Is(err, monitors.ErrMonitorDisabled) {
				logp.L().Info("skipping disabled monitor: %s", err)
				continue // don't stop loading monitors just because they're disabled
			}

			return nil, fmt.Errorf("could not create monitor: %w", err)
		}

		created.Start()
		runners = append(runners, created)
	}

	stop = func() {
		for _, runner := range runners {
			runner.Stop()
		}
	}
	return stop, nil
}

// RunCentralMgmtMonitors loads any central management configured configs.
func (bt *Heartbeat) RunCentralMgmtMonitors(b *beat.Beat) {
	mons := cfgfile.NewRunnerList(management.DebugK, bt.dynamicFactory, b.Publisher)
	reload.Register.MustRegisterList(b.Info.Beat+".monitors", mons)
	inputs := cfgfile.NewRunnerList(management.DebugK, bt.dynamicFactory, b.Publisher)
	reload.Register.MustRegisterList("inputs", inputs)
}

// RunReloadableMonitors runs the `heartbeat.config.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunReloadableMonitors() (err error) {
	// Check monitor configs
	if err := bt.monitorReloader.Check(bt.dynamicFactory); err != nil {
		logp.Error(fmt.Errorf("error loading reloadable monitors: %w", err))
	}

	// Execute the monitor
	go bt.monitorReloader.Run(bt.dynamicFactory)

	return nil
}

// makeAutodiscover creates an autodiscover object ready to be started.
func (bt *Heartbeat) makeAutodiscover(b *beat.Beat) (*autodiscover.Autodiscover, error) {
	ad, err := autodiscover.NewAutodiscover(
		"heartbeat",
		b.Publisher,
		bt.dynamicFactory,
		autodiscover.QueryConfig(),
		bt.config.Autodiscover,
		b.Keystore,
	)
	if err != nil {
		return nil, err
	}
	return ad, nil
}

// Stop stops the beat.
func (bt *Heartbeat) Stop() {
	close(bt.done)
}
