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
	"context"
	"errors"
	"fmt"
	"sync"

	"syscall"
	"time"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/hbregistry"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	_ "github.com/elastic/beats/v7/heartbeat/security"
	"github.com/elastic/beats/v7/heartbeat/tracer"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
)

// Heartbeat represents the root datastructure of this beat.
type Heartbeat struct {
	done     chan struct{}
	stopOnce sync.Once
	// config is used for iterating over elements of the config.
	config             *config.Config
	scheduler          *scheduler.Scheduler
	monitorReloader    *cfgfile.Reloader
	monitorFactory     *monitors.RunnerFactory
	autodiscover       *autodiscover.Autodiscover
	replaceStateLoader func(sl monitorstate.StateLoader)
	trace              tracer.Tracer
}

// New creates a new heartbeat.
func New(b *beat.Beat, rawConfig *conf.C) (beat.Beater, error) {
	parsedConfig := config.DefaultConfig()
	if err := rawConfig.Unpack(&parsedConfig); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Check if any of these can prevent using states client
	stateLoader, replaceStateLoader := monitorstate.AtomicStateLoader(monitorstate.NilStateLoader)
	if b.Config.Output.Name() == "elasticsearch" && !b.Manager.Enabled() {
		// Connect to ES and setup the State loader if the output is not managed by agent
		// Note this, intentionally, blocks until connected or max attempts reached
		esClient, err := makeESClient(b.Config.Output.Config(), 3, 2*time.Second)
		if err != nil {
			if parsedConfig.RunOnce {
				return nil, fmt.Errorf("run_once mode fatal error: %w", err)
			} else {
				logp.L().Warnf("skipping monitor state management: %w", err)
			}
		} else {
			replaceStateLoader(monitorstate.MakeESLoader(esClient, monitorstate.DefaultDataStreams, parsedConfig.RunFrom))
		}
	} else if b.Manager.Enabled() {
		stateLoader, replaceStateLoader = monitorstate.DeferredStateLoader(monitorstate.NilStateLoader, 15*time.Second)
	}

	var trace tracer.Tracer = tracer.NewNoopTracer()
	stConfig := parsedConfig.SocketTrace
	if stConfig != nil {
		// Note this, intentionally, blocks until connected to the trace endpoint
		var err error
		logp.L().Infof("Setting up sock tracer at %s (wait: %s)", stConfig.Path, stConfig.Wait)
		trace, err = tracer.NewSockTracer(stConfig.Path, stConfig.Wait)
		if err != nil {
			logp.L().Fatalf("could not connect to socket trace at path %s after %s timeout: %w", stConfig.Path, stConfig.Wait, err)
		}
	}

	limit := parsedConfig.Scheduler.Limit
	schedLocationName := parsedConfig.Scheduler.Location
	if schedLocationName == "" {
		schedLocationName = "Local"
	}
	location, err := time.LoadLocation(schedLocationName)
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
		done:               make(chan struct{}),
		config:             parsedConfig,
		scheduler:          sched,
		replaceStateLoader: replaceStateLoader,
		// monitorFactory is the factory used for creating all monitor instances,
		// wiring them up to everything needed to actually execute.
		monitorFactory: monitors.NewFactory(monitors.FactoryParams{
			BeatInfo:              b.Info,
			AddTask:               sched.Add,
			StateLoader:           stateLoader,
			PluginsReg:            plugin.GlobalPluginsReg,
			PipelineClientFactory: pipelineClientFactory,
			BeatRunFrom:           parsedConfig.RunFrom,
		}),
		trace: trace,
	}
	runFromID := "<unknown location>"
	if parsedConfig.RunFrom != nil {
		runFromID = parsedConfig.RunFrom.ID
	}
	logp.L().Infof("heartbeat starting, running from: %v", runFromID)
	return bt, nil
}

// Run executes the beat.
func (bt *Heartbeat) Run(b *beat.Beat) error {
	err := bt.trace.Write("start\n")
	if err != nil {
		logp.L().Errorf("could not start trace: %s", err)
	}
	defer func() {
		err := bt.trace.Close()
		if err != nil {
			logp.L().Errorf("could not close trace: %s", err)
		}
	}()

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

	err = bt.trace.Write("stop")
	if err != nil {
		logp.L().Errorf("could not write trace stop event: %s", err)
	}
	logp.L().Info("Shutting down.")
	return nil
}

// RunStaticMonitors runs the `heartbeat.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunStaticMonitors(b *beat.Beat) (stop func(), err error) {
	runners := make([]cfgfile.Runner, 0, len(bt.config.Monitors))
	for _, cfg := range bt.config.Monitors {
		created, err := bt.monitorFactory.Create(b.Publisher, cfg)
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
	// Register output reloader for managed outputs
	b.OutputConfigReloader = reload.ReloadableFunc(func(r *reload.ConfigWithMeta) error {
		// Do not return error here, it will prevent libbeat output from processing the same event
		if r == nil {
			return nil
		}
		outCfg := conf.Namespace{}
		//nolint:nilerr // we are intentionally ignoring specific errors here
		if err := r.Config.Unpack(&outCfg); err != nil || outCfg.Name() != "elasticsearch" {
			return nil
		}

		// Backoff panics with 0 duration, set to smallest unit
		esClient, err := makeESClient(outCfg.Config(), 1, 1*time.Nanosecond)
		if err != nil {
			logp.L().Warnf("skipping monitor state management during managed reload: %w", err)
		} else {
			bt.replaceStateLoader(monitorstate.MakeESLoader(esClient, monitorstate.DefaultDataStreams, bt.config.RunFrom))
		}

		return nil
	})

	inputs := cfgfile.NewRunnerList(management.DebugK, bt.monitorFactory, b.Publisher)
	reload.RegisterV2.MustRegisterInput(inputs)
}

// RunReloadableMonitors runs the `heartbeat.config.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunReloadableMonitors() (err error) {
	// Check monitor configs
	if err := bt.monitorReloader.Check(bt.monitorFactory); err != nil {
		logp.L().Error(fmt.Errorf("error loading reloadable monitors: %w", err))
	}

	// Execute the monitor
	go bt.monitorReloader.Run(bt.monitorFactory)

	return nil
}

// makeAutodiscover creates an autodiscover object ready to be started.
func (bt *Heartbeat) makeAutodiscover(b *beat.Beat) (*autodiscover.Autodiscover, error) {
	ad, err := autodiscover.NewAutodiscover(
		"heartbeat",
		b.Publisher,
		bt.monitorFactory,
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
	bt.stopOnce.Do(func() { close(bt.done) })
}

func makeESClient(cfg *conf.C, attempts int, wait time.Duration) (*eslegclient.Connection, error) {
	var (
		esClient *eslegclient.Connection
		err      error
	)

	// ES client backoff
	connectDelay := backoff.NewEqualJitterBackoff(
		context.Background().Done(),
		wait,
		wait,
	)

	for i := 0; i < attempts; i++ {
		esClient, err = eslegclient.NewConnectedClient(cfg, "Heartbeat")
		if err == nil {
			connectDelay.Reset()
			return esClient, nil
		} else {
			connectDelay.Wait()
		}
	}

	return nil, fmt.Errorf("could not establish states loader connection after %d attempts, with %s delay", attempts, wait)
}
