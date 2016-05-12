package beater

import (
	"expvar"
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
	config  *Config          // Metricbeat specific configuration data.
	modules []*ModuleWrapper // Active list of modules.
	client  publisher.Client // Publisher client.
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
	bt.modules, err = NewModuleWrappers(bt.config.Modules, mb.Registry)
	return err
}

// Run starts the workers for Metricbeat and blocks until Stop is called
// and the workers complete. Each host associated with a MetricSet is given its
// own goroutine for fetching data. The ensures that each host is isolated so
// that a single unresponsive host cannot inadvertently block other hosts
// within the same Module and MetricSet from collection.
func (bt *Metricbeat) Run(b *beat.Beat) error {
	// Start each module.
	var cs []<-chan common.MapStr
	for _, mw := range bt.modules {
		c := mw.Start(bt.done)
		cs = append(cs, c)
	}

	// Consume data from all modules and publish it. When the modules stop they
	// close their output channels. When all the modules' channels are closed
	// PublishChannels exit.
	bt.client = b.Publisher.Connect()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		PublishChannels(bt.client, cs...)
	}()

	// Wait for PublishChannels to stop publishing.
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
	bt.client.Close()
}
