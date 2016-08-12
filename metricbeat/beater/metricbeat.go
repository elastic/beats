package beater

import (
	"expvar"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

// Metricbeat implements the Beater interface for metricbeat.
type Metricbeat struct {
	modules []*ModuleWrapper // Active list of modules.
}

// New creates and returns a new Metricbeat instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	// List all registered modules and metricsets.
	logp.Info("%s", mb.Registry.String())

	config := Config{}
	err := rawConfig.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	modules, err := NewModuleWrappers(config.Modules, mb.Registry)
	if err != nil {
		return nil, err
	}

	mb := &Metricbeat{
		modules: modules,
	}
	return mb, nil
}

// Run starts the workers for Metricbeat and blocks until Stop is called
// and the workers complete. Each host associated with a MetricSet is given its
// own goroutine for fetching data. The ensures that each host is isolated so
// that a single unresponsive host cannot inadvertently block other hosts
// within the same Module and MetricSet from collection.
func (bt *Metricbeat) Run(b *beat.Beat) error {
	defer dumpMetrics()

	client := b.Publisher.Connect()
	b.Done.OnStop.Close(client)

	// Start each module.
	var cs []<-chan common.MapStr
	for _, mw := range bt.modules {
		c := mw.Start(b.Done.C)
		cs = append(cs, c)
	}

	// Consume data from all modules and publish it. When the modules stop they
	// close their output channels. When all the modules' channels are closed
	// PublishChannels exit.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		PublishChannels(client, cs...)
	}()

	// Wait for PublishChannels to stop publishing.
	wg.Wait()
	return nil
}

// dumpMetrics is used to log metrics on shutdown.
func dumpMetrics() {
	logp.Info("Dumping runtime metrics...")
	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key != "memstats" {
			logp.Info("%s=%s", kv.Key, kv.Value.String())
		}
	})
}
