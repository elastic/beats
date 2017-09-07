package cfgfile

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/joeshaw/multierror"

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
	registry      *Registry
	config        DynamicConfig
	runnerFactory RunnerFactory
	path          string
	done          chan struct{}
	wg            sync.WaitGroup
	watcher       *GlobWatcher
}

// NewReloader creates new Reloader instance for the given config
func NewReloader(cfg *common.Config, runnerFactory RunnerFactory) (*Reloader, error) {
	if !cfg.Enabled() {
		// Return nil reloader if it's not enabled, it will do nothing and return no errors
		return nil, nil
	}

	config := DefaultDynamicConfig
	cfg.Unpack(&config)

	path := config.Path
	if !filepath.IsAbs(path) {
		path = paths.Resolve(paths.Config, path)
	}

	reloader := Reloader{
		registry:      NewRegistry(),
		config:        config,
		runnerFactory: runnerFactory,
		path:          path,
		done:          make(chan struct{}),
		watcher:       NewGlobWatcher(path),
	}

	// Scan for current configs and fill regisry
	startRunners, _, _, err := reloader.scan(false)
	for _, r := range startRunners {
		reloader.registry.Add(r.ID(), r)
	}

	// Ignore errors if reload is enabled, they may be solved by the user later on
	if err != nil && !config.Reload.Enabled {
		return nil, err
	}

	return &reloader, nil
}

func (rl *Reloader) Runners() []Runner {
	if rl == nil {
		return nil
	}

	var runners []Runner
	for _, runner := range rl.registry.List {
		runners = append(runners, runner)
	}
	return runners
}

func (rl *Reloader) ReloadEnabled() bool {
	if rl == nil {
		return false
	}
	return rl.config.Reload.Enabled
}

// Run runs the reloader
func (rl *Reloader) Run() {
	if rl == nil {
		return
	}

	logp.Info("Config reloader started")

	rl.wg.Add(1)
	defer rl.wg.Done()

	// Stop all running modules when method finishes
	defer rl.stopRunners(rl.registry.CopyList())

	// Start all runners in the registry
	rl.startRunners(rl.registry.List)

	// If reloading is disabled we are done
	if !rl.config.Reload.Enabled {
		<-rl.done
		return
	}

	// Manage reloading:
	var startList, stopList map[uint64]Runner
	overwriteUpate := true
	for {
		select {
		case <-rl.done:
			logp.Info("Dynamic config reloader stopped")
			return

		case <-time.After(rl.config.Reload.Period):
			startList, stopList, overwriteUpate, _ = rl.scan(overwriteUpate)
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

func (rl *Reloader) scan(overwriteUpdate bool) (map[uint64]Runner, map[uint64]Runner, bool, error) {
	debugf("Scan for new config files")
	configReloads.Add(1)

	files, updated, err := rl.watcher.Scan()
	if err != nil {
		return nil, nil, false, err
	}

	// no file changes
	if !updated && !overwriteUpdate {
		return nil, nil, false, nil
	}

	// Load all config objects
	configs, _ := rl.loadConfigs(files)

	debugf("Number of module configs found: %v", len(configs))

	startList := map[uint64]Runner{}
	stopList := rl.registry.CopyList()
	errs := multierror.Errors{}

	for _, c := range configs {
		// Only add configs to startList which are enabled
		if !c.Enabled() {
			continue
		}

		runner, err := rl.runnerFactory.Create(c)
		if err != nil {
			// Make sure the next run also updates because some runners were not properly loaded
			overwriteUpdate = true
			errs = append(errs, err)

			// In case prospector already is running, do not stop it
			if runner != nil && rl.registry.Has(runner.ID()) {
				debugf("Remove module from stoplist: %v", runner.ID())
				delete(stopList, runner.ID())
			} else {
				logp.Err("Error creating module: %s", err)
			}
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

	return startList, stopList, overwriteUpdate, errs.Err()
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
	if rl == nil {
		return
	}

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
