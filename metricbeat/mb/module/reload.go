package module

import (
	"expvar"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	configReloads = expvar.NewInt("metricbeat.config.reloads")
	moduleStarts  = expvar.NewInt("metricbeat.config.module.starts")
	moduleStops   = expvar.NewInt("metricbeat.config.module.stops")
	moduleRunning = expvar.NewInt("metricbeat.config.module.running")
)

// Reloader is used to register and reload modules
type Reloader struct {
	registry *registry
	config   cfgfile.ReloadConfig
	client   func() publisher.Client
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewReloader creates new Reloader instance for the given config
func NewReloader(cfg *common.Config, p publisher.Publisher) *Reloader {

	config := cfgfile.DefaultReloadConfig
	cfg.Unpack(&config)

	return &Reloader{
		registry: newRunningRegistry(),
		config:   config,
		client:   p.Connect,
		done:     make(chan struct{}),
	}
}

// Run runs the reloader
func (r *Reloader) Run() {

	logp.Info("Config reloader started")

	r.wg.Add(1)
	defer r.wg.Done()

	// Stop all running modules when method finishes
	defer r.stopModules(r.registry.CopyList())

	path := r.config.Path
	if !filepath.IsAbs(path) {
		path = paths.Resolve(paths.Config, path)
	}

	gw := cfgfile.NewGlobWatcher(path)

	for {
		select {
		case <-r.done:
			logp.Info("Dynamic config reloader stopped")
			return
		case <-time.After(r.config.Period):

			debugf("Scan for new config files")
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
				c, err := cfgfile.LoadList(file)
				if err != nil {
					logp.Err("Error loading config: %s", err)
					continue
				}

				configs = append(configs, c...)
			}

			// Check which configs do not exist anymore
			s, err := NewWrappers(configs, mb.Registry)
			if err != nil {
				if err != mb.ErrAllModulesDisabled && err != mb.ErrEmptyConfig {
					// Continuing as only some modules could have an error
					logp.Err("Error creating modules: %s", err)
				}
			}

			debugf("Number of module wrappers created: %v", len(s))

			var startList []*Wrapper
			stopList := r.registry.CopyList()

			for _, w := range s {

				// Only add modules to startlist which are enabled
				if !w.Config().Enabled {
					continue
				}

				debugf("Remove from stoplist: %v", w.Hash())
				delete(stopList, w.Hash())

				// As module already exist, it must be removed from the stop list and not started
				if !r.registry.Has(w.Hash()) {
					debugf("Add to startlist: %v", w.Hash())
					startList = append(startList, w)
					continue
				}
			}

			r.stopModules(stopList)
			r.startModules(startList)
		}
	}
}

// Stop stops the reloader and waits for all modules to properly stop
func (r *Reloader) Stop() {
	close(r.done)
	r.wg.Wait()
}

func (r *Reloader) startModules(list []*Wrapper) {

	logp.Info("Starting %v modules ...", len(list))
	for _, mw := range list {
		mr := NewRunner(r.client, mw)
		mr.Start()
		r.registry.Add(mw.Hash(), mr)
		moduleStarts.Add(1)
		moduleRunning.Add(1)
		debugf("New Module Started: %v", mw.Hash())
	}
}

func (r *Reloader) stopModules(list map[uint64]Runner) {
	logp.Info("Stopping %v modules ...", len(list))

	wg := sync.WaitGroup{}
	for hash, w := range list {
		wg.Add(1)
		// Stop modules in parallel
		func() {
			defer wg.Done()
			w.Stop()
			r.registry.Remove(hash)
			moduleStops.Add(1)
			moduleRunning.Add(-1)
			debugf("Module stopped: %v", hash)
		}()
	}

	wg.Wait()
}
