/*
Package beater provides the implementation of the libbeat Beater interface for
Metricbeat. The main event loop is implemented in this package. The public
interfaces used in implementing Modules and MetricSets are defined in the
github.com/elastic/beats/metricbeat/mb package.

Metricbeat collects metric sets from different modules.

Each event created has the following format:

	curl -XPUT http://localhost:9200/metricbeat/metricsets -d
	{
		"metriset": metricsetName,
		"module": moduleName,
		"moduleName-metricSetName": {
			"metric1": "value",
			"metric2": "value",
			"metric3": "value",
			"nestedmetric": {
				"metric4": "value"
			}
		},
		"@timestamp": timestamp
	}

All documents are stored in one index called metricbeat. It is important to use
an independent namespace for each MetricSet to prevent type conflicts. Also all
values are stored under the same type "metricsets".
*/
package beater

import (
	"expvar"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	done    chan struct{}    // Channel used to initiate shutdown.
	config  *Config          // Metricbeat specific configuration data.
	modules []*moduleWrapper // Active list of modules.
}

// New creates and returns a new Metricbeat instance.
func New() *Metricbeat {
	return &Metricbeat{}
}

// Config unpacks the Metricbeat specific configuration data.
func (bt *Metricbeat) Config(b *beat.Beat) error {
	// List all registered modules and metricsets.
	logp.Info("%s", mb.Registry.String())

	bt.config = &Config{}
	err := b.RawConfig.Unpack(bt.config)
	if err != nil {
		return errors.Wrap(err, "error reading configuration file")
	}

	return nil
}

// Setup initializes the Modules and MetricSets that are defined in the
// Metricbeat configuration.
func (bt *Metricbeat) Setup(b *beat.Beat) error {
	var err error
	bt.done = make(chan struct{})
	bt.modules, err = newModuleWrappers(bt.config.Modules, mb.Registry, b.Publisher)
	return err
}

// Run starts the workers for Metricbeat and blocks until Stop is called
// and the workers complete. Each host associated with a MetricSet is given its
// own goroutine for fetching data. The ensures that each host is isolated so
// that a single unresponsive host cannot inadvertently block other hosts
// within the same Module and MetricSet from collection.
func (bt *Metricbeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup
	for _, mw := range bt.modules {
		wg.Add(len(mw.metricSets))
		for _, msw := range mw.metricSets {
			go msw.startFetching(bt.done, &wg)
		}
	}

	wg.Wait()
	return nil
}

// Cleanup performs clean-up after Run completes.
func (bt *Metricbeat) Cleanup(b *beat.Beat) error {
	logp.Info("Dumping runtime metrics...")
	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key != "memstats" {
			logp.Info("%s=%s", kv.Key, kv.Value.String())
		}
	})
	return nil
}

// Stop signals to Metricbeat that it should stop. It closes the "done" channel
// and closes the publisher client associated with each Module.
//
// Stop should only be called a single time. Calling it more than once may
// result in undefined behavior.
func (bt *Metricbeat) Stop() {
	close(bt.done)
	for _, moduleWrapper := range bt.modules {
		moduleWrapper.pubClient.Close()
	}
}
