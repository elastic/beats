package beater

import (
	"sync"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"

	// Add metricbeat default processors
	_ "github.com/elastic/beats/metricbeat/processor/add_kubernetes_metadata"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	done         chan struct{}  // Channel used to initiate shutdown.
	modules      []staticModule // Active list of modules.
	config       Config
	autodiscover *autodiscover.Autodiscover
}

type staticModule struct {
	connector *module.Connector
	module    *module.Wrapper
}

// New creates and returns a new Metricbeat instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	// List all registered modules and metricsets.
	logp.Debug("modules", "%s", mb.Registry.String())

	config := defaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	dynamicCfgEnabled := config.ConfigModules.Enabled() || config.Autodiscover != nil
	if !dynamicCfgEnabled && len(config.Modules) == 0 {
		return nil, mb.ErrEmptyConfig
	}

	var errs multierror.Errors
	var modules []staticModule
	for _, moduleCfg := range config.Modules {
		if !moduleCfg.Enabled() {
			continue
		}

		failed := false

		err := cfgwarn.CheckRemoved5xSettings(moduleCfg, "filters")
		if err != nil {
			errs = append(errs, err)
			failed = true
		}

		connector, err := module.NewConnector(b.Publisher, moduleCfg, nil)
		if err != nil {
			errs = append(errs, err)
			failed = true
		}

		module, err := module.NewWrapper(config.MaxStartDelay, moduleCfg, mb.Registry)
		if err != nil {
			errs = append(errs, err)
			failed = true
		}

		if failed {
			continue
		}

		modules = append(modules, staticModule{
			connector: connector,
			module:    module,
		})
	}

	if err := errs.Err(); err != nil {
		return nil, err
	}
	if len(modules) == 0 && !dynamicCfgEnabled {
		return nil, mb.ErrAllModulesDisabled
	}

	var adiscover *autodiscover.Autodiscover
	if config.Autodiscover != nil {
		var err error
		factory := module.NewFactory(config.MaxStartDelay, b.Publisher)
		adapter := NewAutodiscoverAdapter(factory)
		adiscover, err = autodiscover.NewAutodiscover("metricbeat", adapter, config.Autodiscover)
		if err != nil {
			return nil, err
		}
	}

	mb := &Metricbeat{
		done:         make(chan struct{}),
		modules:      modules,
		config:       config,
		autodiscover: adiscover,
	}
	return mb, nil
}

// Run starts the workers for Metricbeat and blocks until Stop is called
// and the workers complete. Each host associated with a MetricSet is given its
// own goroutine for fetching data. The ensures that each host is isolated so
// that a single unresponsive host cannot inadvertently block other hosts
// within the same Module and MetricSet from collection.
func (bt *Metricbeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup

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

	if bt.config.ConfigModules.Enabled() {
		moduleReloader := cfgfile.NewReloader(bt.config.ConfigModules)
		factory := module.NewFactory(bt.config.MaxStartDelay, b.Publisher)

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
// under dynamic config settings
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
				m, err := module.NewWrapper(time.Duration(0), conf, mb.Registry)
				if err != nil {
					return nil, errors.Wrap(err, "module initialization error")
				}
				modules = append(modules, m)
			}
		}
	}

	return modules, nil
}
