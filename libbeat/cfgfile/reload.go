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

package cfgfile

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/paths"
)

var (
	// DefaultDynamicConfig provides default behavior for a Runner.
	DefaultDynamicConfig = DynamicConfig{
		Reload: Reload{
			Period:  10 * time.Second,
			Enabled: false,
		},
	}

	debugf = logp.MakeDebug("cfgfile")

	configReloads = monitoring.NewInt(nil, "libbeat.config.reloads")
	moduleStarts  = monitoring.NewInt(nil, "libbeat.config.module.starts")
	moduleStops   = monitoring.NewInt(nil, "libbeat.config.module.stops")
	moduleRunning = monitoring.NewInt(nil, "libbeat.config.module.running")
)

// DynamicConfig loads config files from a given path, allowing to reload new changes
// while running the beat
type DynamicConfig struct {
	// If path is a relative path, it is relative to the ${path.config}
	Path   string `config:"path"`
	Reload Reload `config:"reload"`
}

// Reload defines reload behavior and frequency
type Reload struct {
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}

// RunnerFactory is used for creating of new Runners
type RunnerFactory interface {
	Create(p beat.Pipeline, config *common.Config, meta *common.MapStrPointer) (Runner, error)
}

// ConfigChecker is usually combined with a RunnerFactory for implementations that can check a config
// without a pipeline and metadata.
type ConfigChecker interface {
	CheckConfig(config *common.Config) error
}

// CheckableRunnerFactory is the union of RunnerFactory and ConfigChecker.
type CheckableRunnerFactory interface {
	RunnerFactory
	ConfigChecker
}

// Runner is a simple interface providing a simple way to
// Start and Stop Reloader
type Runner interface {
	// We include fmt.Stringer here because we do log debug messages that must print
	// something for the given Runner. We need Runner implementers to consciously implement a
	// String() method because the default behavior of `%s` is to print everything recursively
	// in a struct, which could cause a race that would cause the race detector to fail.
	// This is something that could be anticipated for the Runner interface specifically, because
	// most runners will use a goroutine that modifies internal state.
	fmt.Stringer
	Start()
	Stop()
}

// Reloader is used to register and reload modules
type Reloader struct {
	pipeline beat.Pipeline
	config   DynamicConfig
	path     string
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewReloader creates new Reloader instance for the given config
func NewReloader(pipeline beat.Pipeline, cfg *common.Config) *Reloader {
	config := DefaultDynamicConfig
	cfg.Unpack(&config)

	path := config.Path
	if !filepath.IsAbs(path) {
		path = paths.Resolve(paths.Config, path)
	}

	return &Reloader{
		pipeline: pipeline,
		config:   config,
		path:     path,
		done:     make(chan struct{}),
	}
}

// Check configs are valid (only if reload is disabled)
func (rl *Reloader) Check(runnerFactory RunnerFactory) error {
	// If config reload is enabled we ignore errors (as they may be fixed afterwards)
	if rl.config.Reload.Enabled {
		return nil
	}

	debugf("Checking module configs from: %s", rl.path)
	gw := NewGlobWatcher(rl.path)

	files, _, err := gw.Scan()
	if err != nil {
		return errors.Wrap(err, "fetching config files")
	}

	// Load all config objects
	configs, err := rl.loadConfigs(files)
	if err != nil {
		return err
	}

	debugf("Number of module configs found: %v", len(configs))

	// Initialize modules
	for _, c := range configs {
		// Only add configs to startList which are enabled
		if !c.Config.Enabled() {
			continue
		}

		if checker, ok := runnerFactory.(ConfigChecker); ok {
			err = checker.CheckConfig(c.Config)
		} else {
			_, err = runnerFactory.Create(rl.pipeline, c.Config, c.Meta)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// Run runs the reloader
func (rl *Reloader) Run(runnerFactory RunnerFactory) {
	logp.Info("Config reloader started")

	list := NewRunnerList("reload", runnerFactory, rl.pipeline)

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer list.Stop()

	gw := NewGlobWatcher(rl.path)

	// If reloading is disable, config files should be loaded immediately
	if !rl.config.Reload.Enabled {
		rl.config.Reload.Period = 0
	}

	overwriteUpdate := true

	for {
		select {
		case <-rl.done:
			logp.Info("Dynamic config reloader stopped")
			return

		case <-time.After(rl.config.Reload.Period):
			debugf("Scan for new config files")
			configReloads.Add(1)

			files, updated, err := gw.Scan()
			if err != nil {
				// In most cases of error, updated == false, so will continue
				// to next iteration below
				logp.Err("Error fetching new config files: %v", err)
			}

			// no file changes
			if !updated && !overwriteUpdate {
				overwriteUpdate = false
				continue
			}

			// Load all config objects
			configs, _ := rl.loadConfigs(files)

			debugf("Number of module configs found: %v", len(configs))

			if err := list.Reload(configs); err != nil {
				// Make sure the next run also updates because some runners were not properly loaded
				overwriteUpdate = true
			}
		}

		// Path loading is enabled but not reloading. Loads files only once and then stops.
		if !rl.config.Reload.Enabled {
			logp.Info("Loading of config files completed.")
			select {
			case <-rl.done:
				logp.Info("Dynamic config reloader stopped")
				return
			}
		}
	}
}

// Load loads configuration files once.
func (rl *Reloader) Load(runnerFactory RunnerFactory) {
	list := NewRunnerList("load", runnerFactory, rl.pipeline)

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer list.Stop()

	gw := NewGlobWatcher(rl.path)

	debugf("Scan for config files")
	files, _, err := gw.Scan()
	if err != nil {
		logp.Err("Error fetching new config files: %v", err)
	}

	// Load all config objects
	configs, _ := rl.loadConfigs(files)

	debugf("Number of module configs found: %v", len(configs))

	if err := list.Reload(configs); err != nil {
		logp.Err("Error loading configuration files: %+v", err)
		return
	}

	logp.Info("Loading of config files completed.")
}

func (rl *Reloader) loadConfigs(files []string) ([]*reload.ConfigWithMeta, error) {
	// Load all config objects
	result := []*reload.ConfigWithMeta{}
	var errs multierror.Errors
	for _, file := range files {
		configs, err := LoadList(file)
		if err != nil {
			errs = append(errs, err)
			logp.Err("Error loading config from file '%s', error %v", file, err)
			continue
		}

		for _, c := range configs {
			result = append(result, &reload.ConfigWithMeta{Config: c})
		}
	}

	return result, errs.Err()
}

// Stop stops the reloader and waits for all modules to properly stop
func (rl *Reloader) Stop() {
	close(rl.done)
	rl.wg.Wait()
}
