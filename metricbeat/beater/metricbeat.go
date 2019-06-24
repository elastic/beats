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
	"sync"

	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/libbeat/management"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"

	// Add autodiscover builders / appenders
	_ "github.com/elastic/beats/metricbeat/autodiscover"

	// Add metricbeat default processors
	_ "github.com/elastic/beats/metricbeat/processor/add_kubernetes_metadata"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	done         chan struct{}  // Channel used to initiate shutdown.
	modules      []staticModule // Active list of modules.
	config       Config
	autodiscover *autodiscover.Autodiscover

	// Options
	moduleOptions []module.Option
}

type staticModule struct {
	connector *module.Connector
	module    *module.Wrapper
}

// Option specifies some optional arguments used for configuring the behavior
// of the Metricbeat framework.
type Option func(mb *Metricbeat)

// WithModuleOptions sets the given module options on the Metricbeat framework
// and these options will be used anytime a new module is instantiated.
func WithModuleOptions(options ...module.Option) Option {
	return func(mb *Metricbeat) {
		mb.moduleOptions = append(mb.moduleOptions, options...)
	}
}

// Creator returns a beat.Creator for instantiating a new instance of the
// Metricbeat framework with the given options.
func Creator(options ...Option) beat.Creator {
	return func(b *beat.Beat, c *common.Config) (beat.Beater, error) {
		return newMetricbeat(b, c, options...)
	}
}

// DefaultCreator returns a beat.Creator for instantiating a new instance of
// Metricbeat framework with the traditional Metricbeat module option of
// module.WithMetricSetInfo.
//
// This is equivalent to calling
//
//     beater.Creator(
//         beater.WithModuleOptions(
//             module.WithMetricSetInfo(),
//         ),
//     )
func DefaultCreator() beat.Creator {
	return Creator(
		WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithServiceName(),
		),
	)
}

// newMetricbeat creates and returns a new Metricbeat instance.
func newMetricbeat(b *beat.Beat, c *common.Config, options ...Option) (*Metricbeat, error) {
	config := defaultConfig
	if err := c.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	dynamicCfgEnabled := config.ConfigModules.Enabled() || config.Autodiscover != nil || b.ConfigManager.Enabled()
	if !dynamicCfgEnabled && len(config.Modules) == 0 {
		return nil, mb.ErrEmptyConfig
	}

	metricbeat := &Metricbeat{
		done:   make(chan struct{}),
		config: config,
	}
	for _, applyOption := range options {
		applyOption(metricbeat)
	}

	// List all registered modules and metricsets.
	logp.Debug("modules", "Available modules and metricsets: %s", mb.Registry.String())

	if b.InSetupCmd {
		// Return without instantiating the metricsets.
		return metricbeat, nil
	}

	moduleOptions := append(
		[]module.Option{module.WithMaxStartDelay(config.MaxStartDelay)},
		metricbeat.moduleOptions...)
	var errs multierror.Errors
	for _, moduleCfg := range config.Modules {
		if !moduleCfg.Enabled() {
			continue
		}

		failed := false

		connector, err := module.NewConnector(b.Publisher, moduleCfg, nil)
		if err != nil {
			errs = append(errs, err)
			failed = true
		}

		module, err := module.NewWrapper(moduleCfg, mb.Registry, moduleOptions...)
		if err != nil {
			errs = append(errs, err)
			failed = true
		}

		if failed {
			continue
		}

		metricbeat.modules = append(metricbeat.modules, staticModule{
			connector: connector,
			module:    module,
		})
	}

	if err := errs.Err(); err != nil {
		return nil, err
	}
	if len(metricbeat.modules) == 0 && !dynamicCfgEnabled {
		return nil, mb.ErrAllModulesDisabled
	}

	if config.Autodiscover != nil {
		var err error
		factory := module.NewFactory(metricbeat.moduleOptions...)
		adapter := autodiscover.NewFactoryAdapter(factory)
		metricbeat.autodiscover, err = autodiscover.NewAutodiscover("metricbeat", b.Publisher, adapter, config.Autodiscover)
		if err != nil {
			return nil, err
		}
	}

	return metricbeat, nil
}

// Run starts the workers for Metricbeat and blocks until Stop is called
// and the workers complete. Each host associated with a MetricSet is given its
// own goroutine for fetching data. The ensures that each host is isolated so
// that a single unresponsive host cannot inadvertently block other hosts
// within the same Module and MetricSet from collection.
func (bt *Metricbeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup

	// Static modules (metricbeat.modules)
	for _, m := range bt.modules {
		client, err := m.connector.Connect()
		if err != nil {
			return err
		}

		r := module.NewRunner(client, m.module)
		r.Start()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			r.Stop()
		}()
	}

	// Centrally managed modules
	factory := module.NewFactory(bt.moduleOptions...)
	modules := cfgfile.NewRunnerList(management.DebugK, factory, b.Publisher)
	reload.Register.MustRegisterList(b.Info.Beat+".modules", modules)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-bt.done
		modules.Stop()
	}()

	// Dynamic file based modules (metricbeat.config.modules)
	if bt.config.ConfigModules.Enabled() {
		moduleReloader := cfgfile.NewReloader(b.Publisher, bt.config.ConfigModules)

		if err := moduleReloader.Check(factory); err != nil {
			return err
		}

		go moduleReloader.Run(factory)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			moduleReloader.Stop()
		}()
	}

	// Autodiscover (metricbeat.autodiscover)
	if bt.autodiscover != nil {
		bt.autodiscover.Start()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			bt.autodiscover.Stop()
		}()
	}

	wg.Wait()
	return nil
}

// Stop signals to Metricbeat that it should stop. It closes the "done" channel
// and closes the publisher client associated with each Module.
//
// Stop should only be called a single time. Calling it more than once may
// result in undefined behavior.
func (bt *Metricbeat) Stop() {
	close(bt.done)
}

// Modules return a list of all configured modules, including anyone present
// under dynamic config settings.
func (bt *Metricbeat) Modules() ([]*module.Wrapper, error) {
	var modules []*module.Wrapper
	for _, m := range bt.modules {
		modules = append(modules, m.module)
	}

	// Add dynamic modules
	if bt.config.ConfigModules.Enabled() {
		config := cfgfile.DefaultDynamicConfig
		bt.config.ConfigModules.Unpack(&config)

		modulesManager, err := cfgfile.NewGlobManager(config.Path, ".yml", ".disabled")
		if err != nil {
			return nil, errors.Wrap(err, "initialization error")
		}

		for _, file := range modulesManager.ListEnabled() {
			confs, err := cfgfile.LoadList(file.Path)
			if err != nil {
				return nil, errors.Wrap(err, "error loading config files")
			}
			for _, conf := range confs {
				m, err := module.NewWrapper(conf, mb.Registry, bt.moduleOptions...)
				if err != nil {
					return nil, errors.Wrap(err, "module initialization error")
				}
				modules = append(modules, m)
			}
		}
	}

	return modules, nil
}
