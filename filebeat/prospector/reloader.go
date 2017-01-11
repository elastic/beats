package prospector

import (
	"expvar"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

var (
	debugr            = logp.MakeDebug("filebeat.reloader")
	configReloads     = expvar.NewInt("filebeat.config.reloads")
	prospectorStarts  = expvar.NewInt("filebeat.config.prospector.dyamic.starts")
	prospectorStops   = expvar.NewInt("filebeat.config.prospector.dyamic.stops")
	prospectorRunning = expvar.NewInt("filebeat.config.prospector.dyamic.running")
)

type ProspectorReloader struct {
	registry  *registry
	config    cfgfile.ReloadConfig
	outlet    Outlet
	done      chan struct{}
	wg        sync.WaitGroup
	registrar *registrar.Registrar
}

func NewProspectorReloader(cfg *common.Config, outlet Outlet, registrar *registrar.Registrar) *ProspectorReloader {

	config := cfgfile.DefaultReloadConfig
	cfg.Unpack(&config)

	return &ProspectorReloader{
		registry:  newRegistry(),
		config:    config,
		outlet:    outlet,
		done:      make(chan struct{}),
		registrar: registrar,
	}
}

func (r *ProspectorReloader) Run() {

	logp.Info("Prospector reloader started")

	r.wg.Add(1)
	defer r.wg.Done()

	// Stop all running prospectors when method finishes
	defer r.stopProspectors(r.registry.CopyList())

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

			debugr("Scan for new config files")

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

			configReloads.Add(1)

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

			debugr("Number of prospectors configs created: %v", len(configs))

			var startList []*Prospector
			stopList := r.registry.CopyList()

			for _, c := range configs {

				// Only add prospectors to startlist which are enabled
				if !c.Enabled() {
					continue
				}

				p, err := NewProspector(c, r.outlet)
				if err != nil {
					logp.Err("Error creating prospector: %s", err)
					continue
				}

				debugr("Remove prospector from stoplist: %v", p.ID)
				delete(stopList, p.ID)

				// As prospector already exist, it must be removed from the stop list and not started
				if !r.registry.Has(p.ID) {
					debugr("Add prospector to startlist: %v", p.ID)
					startList = append(startList, p)
					continue
				}
			}

			r.stopProspectors(stopList)
			r.startProspectors(startList)
		}
	}
}

func (r *ProspectorReloader) startProspectors(prospectors []*Prospector) {
	for _, p := range prospectors {
		err := p.LoadStates(r.registrar.GetStates())
		if err != nil {
			logp.Err("Error loading states for prospector %v: %v", p.ID, err)
			continue
		}
		r.registry.Add(p.ID, p)
		go func(pr *Prospector) {
			prospectorStarts.Add(1)
			prospectorRunning.Add(1)
			defer func() {
				r.registry.Remove(pr.ID)
				logp.Info("Prospector stopped: %v", pr.ID)
			}()
			pr.Run()
		}(p)
	}

}

func (r *ProspectorReloader) stopProspectors(prospectors map[uint64]*Prospector) {
	wg := sync.WaitGroup{}
	for _, p := range prospectors {
		wg.Add(1)
		go func(pr *Prospector) {
			defer wg.Done()
			logp.Debug("reload", "stopping prospector: %v", pr.ID)
			pr.Stop()
			prospectorStops.Add(1)
			prospectorRunning.Add(-1)
		}(p)
	}
	wg.Wait()
}

func (r *ProspectorReloader) Stop() {
	close(r.done)
	// Wait until reloading finished
	r.wg.Wait()

	// Stop all prospectors
	r.stopProspectors(r.registry.CopyList())
}
