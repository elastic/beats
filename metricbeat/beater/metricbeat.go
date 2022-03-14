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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/module"

	// include all metricbeat specific builders
	_ "github.com/elastic/beats/v7/metricbeat/autodiscover/builder/hints"

	// include all metricbeat specific appenders
	_ "github.com/elastic/beats/v7/metricbeat/autodiscover/appender/kubernetes/token"

	// Add metricbeat default processors
	_ "github.com/elastic/beats/v7/metricbeat/processor/add_kubernetes_metadata"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	done         chan struct{}   // Channel used to initiate shutdown.
	runners      []module.Runner // Active list of module runners.
	config       Config
	autodiscover *autodiscover.Autodiscover

	// Options
	moduleOptions []module.Option
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

// WithLightModules enables light modules support
func WithLightModules() Option {
	return func(*Metricbeat) {
		path := paths.Resolve(paths.Home, "module")
		mb.Registry.SetSecondarySource(mb.NewLightModulesSource(path))
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
		WithLightModules(),
		WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithServiceName(),
		),
	)
}

// DefaultTestModulesCreator returns a customized instance of Metricbeat
// where startup delay has been disabled to workaround the fact that
// Modules() will return the static modules (not the dynamic ones)
// with a start delay.
//
// This is equivalent to calling
//
//  beater.Creator(
//		beater.WithLightModules(),
//		beater.WithModuleOptions(
//			module.WithMetricSetInfo(),
//			module.WithMaxStartDelay(0),
//		),
//	)
func DefaultTestModulesCreator() beat.Creator {
	return Creator(
		WithLightModules(),
		WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithMaxStartDelay(0),
		),
	)
}

// newMetricbeat creates and returns a new Metricbeat instance.
func newMetricbeat(b *beat.Beat, c *common.Config, options ...Option) (*Metricbeat, error) {
	config := defaultConfig
	if err := c.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	dynamicCfgEnabled := config.ConfigModules.Enabled() || config.Autodiscover != nil || b.Manager.Enabled()
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

	factory := module.NewFactory(b.Info, moduleOptions...)

	for _, moduleCfg := range config.Modules {
		if !moduleCfg.Enabled() {
			continue
		}

		runner, err := factory.Create(b.Publisher, moduleCfg)
		if err != nil {
			return nil, err
		}

		metricbeat.runners = append(metricbeat.runners, runner)
	}

	if len(metricbeat.runners) == 0 && !dynamicCfgEnabled {
		return nil, mb.ErrAllModulesDisabled
	}

	if config.Autodiscover != nil {
		var err error
		metricbeat.autodiscover, err = autodiscover.NewAutodiscover(
			"metricbeat",
			b.Publisher,
			factory, autodiscover.QueryConfig(),
			config.Autodiscover,
			b.Keystore,
		)
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

	// Static modules (metricbeat.runners)
	for _, r := range bt.runners {
		r.Start()
		wg.Add(1)

		thatRunner := r
		go func() {
			defer wg.Done()
			<-bt.done
			thatRunner.Stop()
		}()
	}

	// Centrally managed modules
	factory := module.NewFactory(b.Info, bt.moduleOptions...)
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

// Modules return a list of all configured modules.
func (bt *Metricbeat) Modules() ([]*module.Wrapper, error) {
	return module.ConfiguredModules(bt.config.Modules, bt.config.ConfigModules, bt.moduleOptions)
}
