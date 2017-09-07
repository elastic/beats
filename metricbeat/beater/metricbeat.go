package beater

import (
	"sync"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"

	// Add metricbeat specific processors
	_ "github.com/elastic/beats/metricbeat/processor/add_kubernetes_metadata"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	done     chan struct{}    // Channel used to initiate shutdown.
	modules  []cfgfile.Runner // Active list of modules.
	config   Config
	reloader *cfgfile.Reloader
	factory  *module.Factory
}

// New creates and returns a new Metricbeat instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	// List all registered modules and metricsets.
	logp.Debug("modules", "%s", mb.Registry.String())

	config := defaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	factory := module.NewFactory(config.MaxStartDelay, b.Publisher)
	var errs multierror.Errors
	var modules []cfgfile.Runner

	// Init static modules
	for _, moduleCfg := range config.Modules {
		if !moduleCfg.Enabled() {
			continue
		}

		if module, err := factory.Create(moduleCfg); err != nil {
			errs = append(errs, err)
		} else {
			modules = append(modules, module)
		}
	}

	// Init config reloader
	reloader, err := cfgfile.NewReloader(config.ConfigModules, factory)
	if err != nil {
		errs = append(errs, err)
	}
	modules = append(modules, reloader.Runners()...)

	if err := errs.Err(); err != nil {
		return nil, err
	}

	if !reloader.ReloadEnabled() {
		if len(modules) == 0 {
			return nil, mb.ErrEmptyConfig
		}
	}
	mb := &Metricbeat{
		done:     make(chan struct{}),
		modules:  modules,
		config:   config,
		factory:  factory,
		reloader: reloader,
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

	for _, module := range bt.modules {
		runner := module
		runner.Start()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			runner.Stop()
		}()
	}

	// Run config reloader
	go bt.reloader.Run()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-bt.done
		bt.reloader.Stop()
	}()

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
		modules = append(modules, m.(module.Runner).Module())
	}

	return modules, nil
}
