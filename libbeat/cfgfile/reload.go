package cfgfile

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/paths"
)

var (
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

type Reload struct {
	Period  time.Duration `config:"period"`
	Enabled bool          `config:"enabled"`
}

type RunnerFactory interface {
	Create(config *common.Config, meta *common.MapStrPointer) (Runner, error)
}

type Runner interface {
	Start()
	Stop()
}

// Reloader is used to register and reload modules
type Reloader struct {
	registry *Registry
	config   DynamicConfig
	path     string
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewReloader creates new Reloader instance for the given config
func NewReloader(cfg *common.Config) *Reloader {
	config := DefaultDynamicConfig
	cfg.Unpack(&config)

	path := config.Path
	if !filepath.IsAbs(path) {
		path = paths.Resolve(paths.Config, path)
	}

	return &Reloader{
		registry: NewRegistry(),
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
		if !c.Enabled() {
			continue
		}
		_, err := runnerFactory.Create(c, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// Run runs the reloader
func (rl *Reloader) Run(runnerFactory RunnerFactory) {
	logp.Info("Config reloader started")

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer rl.stopRunners(rl.registry.CopyList())

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

			startList := map[uint64]Runner{}
			stopList := rl.registry.CopyList()

			for _, c := range configs {

				// Only add configs to startList which are enabled
				if !c.Enabled() {
					continue
				}

				rawCfg := map[string]interface{}{}
				err := c.Unpack(rawCfg)

				if err != nil {
					logp.Err("Unable to unpack config file due to error: %v", err)
					continue
				}

				hash, err := hashstructure.Hash(rawCfg, nil)
				if err != nil {
					// Make sure the next run also updates because some runners were not properly loaded
					overwriteUpdate = true
					debugf("Unable to generate hash for config file %v due to error: %v", c, err)
					continue
				}

				debugf("Remove module from stoplist: %v", hash)
				delete(stopList, hash)

				// As module already exist, it must be removed from the stop list and not started
				if !rl.registry.Has(hash) {
					debugf("Add module to startlist: %v", hash)
					runner, err := runnerFactory.Create(c, nil)
					if err != nil {
						logp.Err("Unable to create runner due to error: %v", err)
						continue
					}
					startList[hash] = runner
				}
			}

			rl.stopRunners(stopList)
			rl.startRunners(startList)
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

func (rl *Reloader) loadConfigs(files []string) ([]*common.Config, error) {
	// Load all config objects
	configs := []*common.Config{}
	var errs multierror.Errors
	for _, file := range files {
		c, err := LoadList(file)
		if err != nil {
			errs = append(errs, err)
			logp.Err("Error loading config: %s", err)
			continue
		}

		configs = append(configs, c...)
	}

	return configs, errs.Err()
}

// Stop stops the reloader and waits for all modules to properly stop
func (rl *Reloader) Stop() {
	close(rl.done)
	rl.wg.Wait()
}

func (rl *Reloader) startRunners(list map[uint64]Runner) {
	if len(list) == 0 {
		return
	}

	logp.Info("Starting %v runners ...", len(list))
	for id, runner := range list {
		runner.Start()
		rl.registry.Add(id, runner)

		moduleStarts.Add(1)
		moduleRunning.Add(1)
		debugf("New runner started: %v", id)
	}
}

func (rl *Reloader) stopRunners(list map[uint64]Runner) {
	if len(list) == 0 {
		return
	}

	logp.Info("Stopping %v runners ...", len(list))

	wg := sync.WaitGroup{}
	for hash, runner := range list {
		wg.Add(1)

		// Stop modules in parallel
		go func(h uint64, run Runner) {
			defer func() {
				moduleStops.Add(1)
				moduleRunning.Add(-1)
				debugf("Runner stopped: %v", h)
				wg.Done()
			}()

			run.Stop()
			rl.registry.Remove(h)
		}(hash, runner)
	}

	wg.Wait()
}
