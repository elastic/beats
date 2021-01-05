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
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/hbregistry"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/management"


	dirCopy "github.com/otiai10/copy"
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
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
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

	scheduler := scheduler.NewWithLocation(limit, hbregistry.SchedulerRegistry, location)

	bt := &Heartbeat{
		done:      make(chan struct{}),
		config:    parsedConfig,
		scheduler: scheduler,
		// dynamicFactory is the factory used for dynamic configs, e.g. autodiscover / reload
		dynamicFactory: monitors.NewFactory(b.Info, scheduler, false),
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

	if b.Manager.Enabled() {
		bt.RunCentralMgmtMonitors(b)
	}

	if bt.config.ConfigMonitors.Enabled() {
		bt.monitorReloader = cfgfile.NewReloader(b.Publisher, bt.config.ConfigMonitors)
		defer bt.monitorReloader.Stop()

		err := bt.RunReloadableMonitors(b)
		if err != nil {
			return err
		}
	}

	if len(bt.config.SyntheticSuites) > 0 {
		err := bt.RunSyntheticSuiteMonitors(b)
		if err != nil {
			return err
		}
	}

	if bt.config.Autodiscover != nil {
		bt.autodiscover, err = bt.makeAutodiscover(b)
		if err != nil {
			return err
		}

		bt.autodiscover.Start()
		defer bt.autodiscover.Stop()
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
	factory := monitors.NewFactory(b.Info, bt.scheduler, true)

	for _, cfg := range bt.config.Monitors {
		created, err := factory.Create(b.Publisher, cfg)
		if err != nil {
			if err == stdfields.ErrPluginDisabled {
				continue // don't stop loading monitors just because they're disabled
			}

			return errors.Wrap(err, "could not create monitor")
		}

		created.Start()
	}
	return nil
}

// RunCentralMgmtMonitors loads any central management configured configs.
func (bt *Heartbeat) RunCentralMgmtMonitors(b *beat.Beat) {
	monitors := cfgfile.NewRunnerList(management.DebugK, bt.dynamicFactory, b.Publisher)
	reload.Register.MustRegisterList(b.Info.Beat+".monitors", monitors)
	inputs := cfgfile.NewRunnerList(management.DebugK, bt.dynamicFactory, b.Publisher)
	reload.Register.MustRegisterList("inputs", inputs)
}

// RunReloadableMonitors runs the `heartbeat.config.monitors` portion of the yaml config if present.
func (bt *Heartbeat) RunReloadableMonitors(b *beat.Beat) (err error) {
	// Check monitor configs
	if err := bt.monitorReloader.Check(bt.dynamicFactory); err != nil {
		logp.Error(errors.Wrap(err, "error loading reloadable monitors"))
	}

	// Execute the monitor
	go bt.monitorReloader.Run(bt.dynamicFactory)

	return nil
}

// Provide hook to define journey list discovery from x-pack
type JourneyLister func(ctx context.Context, suitePath string, params common.MapStr) ([]string, error)

var mainJourneyLister JourneyLister

func RegisterJourneyLister(jl JourneyLister) {
	mainJourneyLister = jl
}

func (bt *Heartbeat) RunSyntheticSuiteMonitors(b *beat.Beat) error {
	// If we are running without XPack this will be nil
	if mainJourneyLister == nil {
		return nil
	}
	for _, rawSuiteCfg := range bt.config.SyntheticSuites {
		suite := &config.SyntheticSuite{}
		err := rawSuiteCfg.Unpack(suite)
		if err != nil {
			logp.Err("could not parse suite config: %s", err)
			continue
		}

		var suiteReloader SuiteReloader

		switch suite.Type {
		case "local":
			localConfig := &config.LocalSyntheticSuite{}
			err := rawSuiteCfg.Unpack(localConfig)
			if err != nil {
				logp.Err("could not parse local synthetic suite: %s", err)
				continue
			}
			suiteReloader, err = NewLocalReloader(localConfig.Path)
			if err != nil {
				logp.Err("could not load local synthetics suite: %s", err)
				continue
			}
		case "zipurl":
			localConfig := &config.LocalSyntheticSuite{}
			err := rawSuiteCfg.Unpack(localConfig)
			if err != nil {
				logp.Err("could not parse zip URL synthetic suite: %s", err)
				continue
			}
		case "github":
			localConfig := &config.LocalSyntheticSuite{}
			err := rawSuiteCfg.Unpack(localConfig)
			if err != nil {
				logp.Err("could not parse github synthetic suite: %s", err)
				continue
			}
		}

		logp.Info("Listing suite %s", suiteReloader.WorkingPath())
		journeyNames, err := mainJourneyLister(context.TODO(), suiteReloader.WorkingPath(), suite.Params)
		if err != nil {
			return err
		}
		factory := monitors.NewFactory(b.Info, bt.scheduler, false)
		for _, name := range journeyNames {
			cfg, err := common.NewConfigFrom(map[string]interface{}{
				"type":         "browser",
				"path":         suiteReloader.WorkingPath(),
				"schedule":     suite.Schedule,
				"params":       suite.Params,
				"journey_name": name,
				"name":         name,
				"id":           name,
			})
			if err != nil {
				return err
			}
			created, err := factory.Create(b.Publisher, cfg)
			if err != nil {
				return errors.Wrap(err, "could not create monitor")
			}
			created.Start()
		}
	}
	return nil
}

// makeAutodiscover creates an autodiscover object ready to be started.
func (bt *Heartbeat) makeAutodiscover(b *beat.Beat) (*autodiscover.Autodiscover, error) {
	autodiscover, err := autodiscover.NewAutodiscover(
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
	return autodiscover, nil
}

// Stop stops the beat.
func (bt *Heartbeat) Stop() {
	close(bt.done)
}

type SuiteReloader interface {
	String() string
	Check() (bool, error)
	WorkingPath() string
}

func NewLocalReloader(origSuitePath string) (*LocalReloader, error) {
	dir, err := ioutil.TempDir("/tmp", "elastic-synthetics-")
	if err != nil {
		return nil, err
	}

	err = dirCopy.Copy(origSuitePath, dir)
	if err != nil {
		return nil, err
	}

	return &LocalReloader{workPath: origSuitePath}, nil
}

type LocalReloader struct {
	origPath string
	workPath string
}

func (l *LocalReloader) String() string {
	return fmt.Sprintf("[Local Synthetics Suite origPath=%s workingPath=%s]", l.origPath, l.WorkingPath())
}

// Only loads once on startup, no rechecks
func (l *LocalReloader) Check() (bool, error) {
	return false, nil
}

func (l *LocalReloader) WorkingPath() string {
	return l.workPath
}

func NewZipURLReloader(url string, headers map[string]string) (*ZipURLReloader, error) {
	return &ZipURLReloader{URL: url, Headers: headers}, nil
}

type ZipURLReloader struct {
	URL string
	Headers map[string]string
	etag string // used to determine if the URL contents has changed
	workingPath string
}

func (z *ZipURLReloader) String() string {
	return fmt.Sprintf("[ZipURL Synthetics suite url=%s workingPath=%s]", z.URL, z.workingPath)
}

func (z *ZipURLReloader) Check() (bool, error) {
	panic("implement me")
}

func (z *ZipURLReloader) WorkingPath() string {
	panic("implement me")
}

