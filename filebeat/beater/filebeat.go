package beater

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/crawler"
	"github.com/elastic/beats/filebeat/publisher"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/filebeat/spooler"
)

// Filebeat is a beater object. Contains all objects needed to run the beat
type Filebeat struct {
	config *cfg.Config
	done   chan struct{}
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
		done:   make(chan struct{}),
		config: &config,
	}
	return fb, nil
}

// Run allows the beater to be run as a beat.
func (fb *Filebeat) Run(b *beat.Beat) error {
	var err error
	config := fb.config

	// Setup registrar to persist state
	registrar, err := registrar.New(config.RegistryFile, nil)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	// Channel from harvesters to spooler
	successLogger := newRegistrarLogger(registrar)
	publisherChan := newPublisherChannel()

	// Publishes event to output
	publisher := publisher.New(config.PublishAsync,
		publisherChan.ch, successLogger, b.Publisher)

	// Init and Start spooler: Harvesters dump events into the spooler.
	spooler, err := spooler.New(config, publisherChan)
	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	crawler, err := crawler.New(
		newSpoolerOutlet(fb.done, spooler, nil),
		config.Prospectors)
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
	defer registrar.Stop()

	// Start publisher
	publisher.Start()
	// Stopping publisher (might potentially drop items)
	defer publisher.Stop()
	defer successLogger.Close()

	// Starting spooler
	spooler.Start()
	// Stopping spooler will flush items
	defer spooler.Stop()
	defer publisherChan.Close()

	err = crawler.Start(registrar.GetStates())
	if err != nil {
		return err
	}
	// Stop crawler -> stop prospectors -> stop harvesters
	defer crawler.Stop()

	// Blocks progressing. As soon as channel is closed, all defer statements come into play
	<-fb.done

	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (fb *Filebeat) Stop() {
	logp.Info("Stopping filebeat")

	// Stop Filebeat
	close(fb.done)
}
