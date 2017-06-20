package cfgfile

import (
	"path/filepath"
	"sync"
	"time"

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
	Create(*common.Config) (Runner, error)
}

type Runner interface {
	Start()
	Stop()
	ID() uint64
}

// Reloader is used to register and reload modules
type Reloader struct {
	registry *Registry
	config   DynamicConfig
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewReloader creates new Reloader instance for the given config
func NewReloader(cfg *common.Config) *Reloader {

	config := DefaultDynamicConfig
	cfg.Unpack(&config)

	return &Reloader{
		registry: NewRegistry(),
		config:   config,
		done:     make(chan struct{}),
	}
}

// Run runs the reloader
func (rl *Reloader) Run(runnerFactory RunnerFactory) {

	logp.Info("Config reloader started")

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer rl.stopRunners(rl.registry.CopyList())

	path := rl.config.Path
	if !filepath.IsAbs(path) {
		path = paths.Resolve(paths.Config, path)
	}

	gw := NewGlobWatcher(path)

	// If reloading is disable, config files should be loaded immidiately
	if !rl.config.Reload.Enabled {
		rl.config.Reload.Period = 0
	}

	overwriteUpate := true

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
			if !updated && !overwriteUpate {
				overwriteUpate = false
				continue
			}

			// Load all config objects
			configs := []*common.Config{}
			for _, file := range files {
				c, err := LoadList(file)
				if err != nil {
					logp.Err("Error loading config: %s", err)
					continue
				}

				configs = append(configs, c...)
			}

			debugf("Number of module configs found: %v", len(configs))

			startList := map[uint64]Runner{}
			stopList := rl.registry.CopyList()

			for _, c := range configs {

				// Only add configs to startList which are enabled
				if !c.Enabled() {
					continue
				}

				runner, err := runnerFactory.Create(c)
				if err != nil {
					// Make sure the next run also updates because some runners were not properly loaded
					overwriteUpate = true

					// In case prospector already is running, do not stop it
					if runner != nil && rl.registry.Has(runner.ID()) {
						debugf("Remove module from stoplist: %v", runner.ID())
						delete(stopList, runner.ID())
					}
					logp.Err("Error creating module: %s", err)
					continue
				}

				debugf("Remove module from stoplist: %v", runner.ID())
				delete(stopList, runner.ID())

				// As module already exist, it must be removed from the stop list and not started
				if !rl.registry.Has(runner.ID()) {
					debugf("Add module to startlist: %v", runner.ID())
					startList[runner.ID()] = runner
					continue
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
