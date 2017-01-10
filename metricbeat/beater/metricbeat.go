package beater

import (
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	done    chan struct{}    // Channel used to initiate shutdown.
	modules []*ModuleWrapper // Active list of modules.
	client  publisher.Client // Publisher client.
	config  Config
}

// New creates and returns a new Metricbeat instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	// List all registered modules and metricsets.
	logp.Info("%s", mb.Registry.String())

	config := DefaultConfig

	err := rawConfig.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	modules, err := NewModuleWrappers(config.Modules, mb.Registry)
	if err != nil {
		// Empty config is fine if dynamic config is enabled
		if !config.ReloadModules.IsEnabled() {
			return nil, err
		} else if err != mb.ErrEmptyConfig && err != mb.ErrAllModulesDisabled {
			return nil, err
		}
	}

	mb := &Metricbeat{
		done:    make(chan struct{}),
		modules: modules,
		config:  config,
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
		r := NewModuleRunner(b.Publisher.Connect, m)
		r.Start()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			r.Stop()
		}()
	}

	if bt.config.ReloadModules.IsEnabled() {
		logp.Warn("EXPERIMENTAL feature dynamic configuration reloading is enabled.")
		configReloader := NewConfigReloader(bt.config.ReloadModules, b.Publisher)
		go configReloader.Run()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			configReloader.Stop()
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
