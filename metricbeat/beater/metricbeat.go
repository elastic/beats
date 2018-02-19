package beater

import (
	"sync"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	mbautodiscover "github.com/elastic/beats/metricbeat/autodiscover"
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
		),
	)
}

// newMetricbeat creates and returns a new Metricbeat instance.
func newMetricbeat(b *beat.Beat, c *common.Config, options ...Option) (*Metricbeat, error) {
	// List all registered modules and metricsets.
	logp.Debug("modules", "%s", mb.Registry.String())

	config := defaultConfig
	if err := c.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	dynamicCfgEnabled := config.ConfigModules.Enabled() || config.Autodiscover != nil
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

	moduleOptions := append(
		[]module.Option{module.WithMaxStartDelay(config.MaxStartDelay)},
		metricbeat.moduleOptions...)
	var errs multierror.Errors
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
		factory := module.NewFactory(b.Publisher, metricbeat.moduleOptions...)
		adapter := mbautodiscover.NewAutodiscoverAdapter(factory)
		metricbeat.autodiscover, err = autodiscover.NewAutodiscover("metricbeat", adapter, config.Autodiscover)
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
		factory := module.NewFactory(b.Publisher, bt.moduleOptions...)

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
