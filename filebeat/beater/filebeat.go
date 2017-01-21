package beater

import (
	"errors"
	"flag"
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/crawler"
	"github.com/elastic/beats/filebeat/fileset"
	"github.com/elastic/beats/filebeat/publisher"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/filebeat/spooler"
)

var (
	once  = flag.Bool("once", false, "Run filebeat only once until all harvesters reach EOF")
	setup = flag.Bool("setup", false, "Run the setup phase for the modules")
)

// Filebeat is a beater object. Contains all objects needed to run the beat
type Filebeat struct {
	config         *cfg.Config
	moduleRegistry *fileset.ModuleRegistry
	done           chan struct{}
}

// New creates a new Filebeat pointer instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	config := cfg.DefaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	moduleRegistry, err := fileset.NewModuleRegistry(config.Modules)
	if err != nil {
		return nil, err
	}

	var moduleProspectors []*common.Config
	if moduleRegistry != nil {
		var err error
		moduleProspectors, err = moduleRegistry.GetProspectorConfigs()
		if err != nil {
			return nil, err
		}
	}

	if err := config.FetchConfigs(); err != nil {
		return nil, err
	}

	// Add prospectors created by the modules
	config.Prospectors = append(config.Prospectors, moduleProspectors...)

	if len(config.Prospectors) == 0 {
		return nil, errors.New("No prospectors defined. What files do you want me to watch?")
	}

	fb := &Filebeat{
		done:           make(chan struct{}),
		config:         &config,
		moduleRegistry: moduleRegistry,
	}
	return fb, nil
}

// Setup is called on user request (the -setup flag) to do the initial Beat setup.
func (fb *Filebeat) Setup(b *beat.Beat) error {
	esConfig := b.Config.Output["elasticsearch"]
	if esConfig == nil || !esConfig.Enabled() {
		return fmt.Errorf("Setup requested but the Elasticsearch output is not configured/enabled")
	}
	esClient, err := elasticsearch.NewConnectedClient(esConfig)
	if err != nil {
		return fmt.Errorf("Error creating ES client: %v", err)
	}
	defer esClient.Close()

	return fb.moduleRegistry.Setup(esClient)
}

// Run allows the beater to be run as a beat.
func (fb *Filebeat) Run(b *beat.Beat) error {
	var err error
	config := fb.config

	if *setup {
		err = fb.Setup(b)
		if err != nil {
			return err
		}
	}

	waitFinished := newSignalWait()
	waitEvents := newSignalWait()

	// count active events for waiting on shutdown
	wgEvents := &sync.WaitGroup{}
	finishedLogger := newFinishedLogger(wgEvents)

	// Setup registrar to persist state
	registrar, err := registrar.New(config.RegistryFile, finishedLogger)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	// Make sure all events that were published in
	registrarChannel := newRegistrarLogger(registrar)

	// Channel from spooler to harvester
	publisherChan := newPublisherChannel()

	// Publishes event to output
	publisher := publisher.New(config.PublishAsync, publisherChan.ch, registrarChannel, b.Publisher)

	// Init and Start spooler: Harvesters dump events into the spooler.
	spooler, err := spooler.New(config, publisherChan)
	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	crawler, err := crawler.New(newSpoolerOutlet(fb.done, spooler, wgEvents), config.Prospectors, *once)
	if err != nil {
		logp.Err("Could not init crawler: %v", err)
		return err
	}

	// The order of starting and stopping is important. Stopping is inverted to the starting order.
	// The current order is: registrar, publisher, spooler, crawler
	// That means, crawler is stopped first.

	// Start the registrar
	err = registrar.Start()
	if err != nil {
		return fmt.Errorf("Could not start registrar: %v", err)
	}
	// Stopping registrar will write last state
	defer registrar.Stop()

	// Start publisher
	publisher.Start()
	// Stopping publisher (might potentially drop items)
	defer func() {
		// Closes first the registrar logger to make sure not more events arrive at the registrar
		// registrarChannel must be closed first to potentially unblock (pretty unlikely) the publisher
		registrarChannel.Close()
		publisher.Stop()
	}()

	// Starting spooler
	spooler.Start()

	// Stopping spooler will flush items
	defer func() {
		// Wait for all events to be processed or timeout
		waitEvents.Wait()

		// Closes publisher so no further events can be sent
		publisherChan.Close()
		// Stopping spooler
		spooler.Stop()
	}()

	err = crawler.Start(registrar)
	if err != nil {
		return err
	}

	// If run once, add crawler completion check as alternative to done signal
	if *once {
		runOnce := func() {
			logp.Info("Running filebeat once. Waiting for completion ...")
			crawler.WaitForCompletion()
			logp.Info("All data collection completed. Shutting down.")
		}
		waitFinished.Add(runOnce)
	}

	// Add done channel to wait for shutdown signal
	waitFinished.AddChan(fb.done)
	waitFinished.Wait()

	// Stop crawler -> stop prospectors -> stop harvesters
	// Note: waiting for crawlers to stop here in order to install wgEvents.Wait
	//       after all events have been enqueued for publishing. Otherwise wgEvents.Wait
	//       or publisher might panic due to concurrent updates.
	crawler.Stop()

	timeout := fb.config.ShutdownTimeout
	// Checks if on shutdown it should wait for all events to be published
	waitPublished := fb.config.ShutdownTimeout > 0 || *once
	if waitPublished {
		// Wait for registrar to finish writing registry
		waitEvents.Add(withLog(wgEvents.Wait,
			"Continue shutdown: All enqueued events being published."))
		// Wait for either timeout or all events having been ACKed by outputs.
		if fb.config.ShutdownTimeout > 0 {
			logp.Info("Shutdown output timer started. Waiting for max %v.", timeout)
			waitEvents.Add(withLog(waitDuration(timeout),
				"Continue shutdown: Time out waiting for events being published."))
		} else {
			waitEvents.AddChan(fb.done)
		}
	}

	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (fb *Filebeat) Stop() {
	logp.Info("Stopping filebeat")

	// Stop Filebeat
	close(fb.done)
}
