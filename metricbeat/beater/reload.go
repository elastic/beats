package beater

import (
	"expvar"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	debugr        = logp.MakeDebug("reloader")
	configReloads = expvar.NewInt("metricbeat.config.reloads")
	moduleStarts  = expvar.NewInt("metricbeat.config.module.starts")
	moduleStops   = expvar.NewInt("metricbeat.config.module.stops")
	moduleRunning = expvar.NewInt("metricbeat.config.module.running")
)

type ConfigReloader struct {
	registry *runningRegistry
	config   ModulesReloadConfig
	client   func() publisher.Client
	done     chan struct{}
	wg       sync.WaitGroup
}

type runningRegistry struct {
	sync.Mutex
	List map[uint64]ModuleRunner
}

func NewConfigReloader(config ModulesReloadConfig, p publisher.Publisher) *ConfigReloader {

	return &ConfigReloader{
		registry: newRunningRegistry(),
		config:   config,
		client:   p.Connect,
		done:     make(chan struct{}),
	}
}

func (r *ConfigReloader) Run() {

	logp.Info("Config reloader started")

	r.wg.Add(1)
	defer r.wg.Done()

	// Stop all running modules when method finishes
	defer r.StopModules(r.registry.CopyList())

	path := r.config.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(cfgfile.GetPathConfig(), path)
	}

	gw := NewGlobWatcher(path)

	for {
		select {
		case <-r.done:
			logp.Info("Dynamic config reloader stopped")
			return
		case <-time.After(r.config.Period):

			debugr("Scan for new config files")
			configReloads.Add(1)

			files, updated, err := gw.Scan()
			if err != nil {
				// In most cases of error, updated == false, so will continue
				// to next iteration below
				logp.Err("Error fetching new config files: %v", err)
			}

			// no file changes
			if !updated {
				continue
			}

			// Load all config objects
			configs := []*common.Config{}
			for _, file := range files {
				c, err := LoadConfigs(file)
				if err != nil {
					logp.Err("Error loading config: %s", err)
					continue
				}

				configs = append(configs, c...)
			}

			// Check which configs do not exist anymore
			s, err := NewModuleWrappers(configs, mb.Registry)
			if err != nil {
				if err != mb.ErrAllModulesDisabled && err != mb.ErrEmptyConfig {
					// Continuing as only some modules could have an error
					logp.Err("Error creating modules: %s", err)
				}
			}

			debugr("Number of module wrappers created: %v", len(s))

			var startList []*ModuleWrapper
			stopList := r.registry.CopyList()

			for _, w := range s {

				// Only add modules to startlist which are enabled
				if !w.Config().Enabled {
					continue
				}

				debugr("Remove from stoplist: %v", w.Hash())
				delete(stopList, w.Hash())

				// As module already exist, it must be removed from the stop list and not started
				if !r.registry.Has(w.Hash()) {
					debugr("Add to startlist: %v", w.Hash())
					startList = append(startList, w)
					continue
				}
			}

			r.StopModules(stopList)
			r.StartModules(startList)
		}
	}
}

func (r *ConfigReloader) Stop() {
	close(r.done)
	r.wg.Wait()
}

func (r *ConfigReloader) StartModules(list []*ModuleWrapper) {

	logp.Info("Starting %v modules ...", len(list))
	for _, w := range list {
		go func(mw *ModuleWrapper) {
			mr := NewModuleRunner(r.client, mw)
			mr.Start()
			r.registry.Add(mw.Hash(), mr)
			moduleStarts.Add(1)
			moduleRunning.Add(1)
			debugr("New Module Started: %v", mw.Hash())
		}(w)
	}
}

func (r *ConfigReloader) StopModules(list map[uint64]ModuleRunner) {
	logp.Info("Stopping %v modules ...", len(list))

	for hash, w := range list {
		w.Stop()
		r.registry.Remove(hash)
		moduleStops.Add(1)
		moduleRunning.Add(-1)
		debugr("Module stopped: %v", hash)
	}
}

// LoadConfigs loads the configs data from the given file
func LoadConfigs(file string) ([]*common.Config, error) {
	debugr("Load config from file: %s", file)
	rawConfig, err := common.LoadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Invalid config: %s", err)
	}

	var c []*common.Config
	err = rawConfig.Unpack(&c)
	if err != nil {
		return nil, fmt.Errorf("Error reading configuration from file %s: %s", file, err)
	}

	return c, nil
}

func newRunningRegistry() *runningRegistry {
	return &runningRegistry{
		List: map[uint64]ModuleRunner{},
	}
}

func (r *runningRegistry) Add(hash uint64, m ModuleRunner) {
	r.Lock()
	defer r.Unlock()
	r.List[hash] = m
}

func (r *runningRegistry) Remove(hash uint64) {
	r.Lock()
	defer r.Unlock()
	delete(r.List, hash)
}

func (r *runningRegistry) Has(hash uint64) bool {
	r.Lock()
	defer r.Unlock()

	_, ok := r.List[hash]
	return ok
}

func (r *runningRegistry) CopyList() map[uint64]ModuleRunner {
	r.Lock()
	defer r.Unlock()

	// Create a copy of the list
	list := map[uint64]ModuleRunner{}
	for k, v := range r.List {
		list[k] = v
	}
	return list
}
