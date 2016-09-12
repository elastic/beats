package beater

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/crawler"
	"github.com/elastic/beats/filebeat/publisher"
	"github.com/elastic/beats/filebeat/registrar"
	spo "github.com/elastic/beats/filebeat/spooler"
)

// Filebeat is a beater object. Contains all objects needed to run the beat
type Filebeat struct {
	config  *cfg.Config
	sigWait *signalWait
	done    chan struct{}
}

// New creates a new Filebeat pointer instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	config := cfg.DefaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}
	if err := config.FetchConfigs(); err != nil {
		return nil, err
	}

	fb := &Filebeat{
		done:    make(chan struct{}),
		sigWait: newSignalWait(),
		config:  &config,
	}
	return fb, nil
}

// Run allows the beater to be run as a beat.
func (fb *Filebeat) Run(b *beat.Beat) error {
	var err error
	config := fb.config

	var publisherEventsWg *sync.WaitGroup // count active events for waiting on shutdown
	var publisherLogger publisher.Logger

	if fb.config.ShutdownTimeout > 0 {
		publisherEventsWg = &sync.WaitGroup{}
		publisherLogger = publisher.NewLog(publisherEventsWg)
	}

	// Setup registrar to persist state
	registrar, err := registrar.New(config.RegistryFile, publisherLogger)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	// Logger for publisher to log sucessfully sent events to registrar
	registrarLogger := registrar.GetLogger()

	// Output to send events to publisher
	publisherOutput := publisher.NewOutput()

	// Creates publisher to send events to output
	publisher := publisher.New(config.PublishAsync, publisherOutput.GetChannel(), registrarLogger, b.Publisher)

	// Init and Start spooler: Harvesters dump events into the spooler.
	spooler, err := spo.New(config, publisherOutput)
	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	spoolerOutput := spo.NewOutput(fb.done, spooler, nil)
	crawler, err := crawler.New(spoolerOutput, config.Prospectors)
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
		logp.Err("Could not start registrar: %v", err)
	}
	// Stopping registrar will write last state
	defer func() {
		// Stop logger itself
		registrar.Stop()
	}()

	// Start publisher
	publisher.Start()
	// Stopping publisher (might potentially drop items)
	defer func() {
		// Stop logger to make sure not new events are added. To make sure registrar
		// never blocks publisher shutdown, this happens before publisher stop.
		registrarLogger.Close()
		// Stopping publisher itself
		publisher.Stop()
	}()

	// Starting spooler
	spooler.Start()

	// Stopping spooler will flush items
	defer func() {
		// With harvesters being stopped, optionally wait for all enqueued events being
		// published and written by registrar before continuing shutdown.
		fb.sigWait.Wait()

		// Publisher output must be closed before spooler shutdown as otherwise spooler could be blocked
		publisherOutput.Close()
		spooler.Stop()
	}()

	err = crawler.Start(registrar.GetStates())
	if err != nil {
		return err
	}
	// Stop crawler -> stop prospectors -> stop harvesters
	defer crawler.Stop()

	// Blocks progressing. As soon as channel is closed, all defer statements come into play
	<-fb.done

	if fb.config.ShutdownTimeout > 0 {
		// Wait for either timeout or all events having been ACKed by outputs.
		fb.sigWait.Add(publisherEventsWg.Wait)
		fb.sigWait.AddTimeout(fb.config.ShutdownTimeout)
	}

	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (fb *Filebeat) Stop() {
	logp.Info("Stopping filebeat")

	// Stop Filebeat
	close(fb.done)
}
