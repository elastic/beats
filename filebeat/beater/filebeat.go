package beater

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/crawler"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/publish"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/filebeat/spooler"
)

// Filebeat is a beater object. Contains all objects needed to run the beat
type Filebeat struct {
	config *cfg.Config
	done   chan struct{}
}

// New creates a new Filebeat pointer instance.
func New() *Filebeat {
	return &Filebeat{}
}

// Config setups up the filebeat configuration by fetch all additional config files
func (fb *Filebeat) Config(b *beat.Beat) error {

	// Load Base config
	err := b.RawConfig.Unpack(&fb.config)
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	// Check if optional config_dir is set to fetch additional prospector config files
	fb.config.FetchConfigs()

	return nil
}

// Setup applies the minimum required setup to a new Filebeat instance for use.
func (fb *Filebeat) Setup(b *beat.Beat) error {
	fb.done = make(chan struct{})
	return nil
}

// Run allows the beater to be run as a beat.
func (fb *Filebeat) Run(b *beat.Beat) error {

	var err error
	config := fb.config.Filebeat

	// Setup registrar to persist state
	registrar, err := registrar.New(config.RegistryFile)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	// Channel from harvesters to spooler
	publisherChan := make(chan []*input.FileEvent, 1)

	// Publishes event to output
	publisher := publish.New(config.PublishAsync,
		publisherChan, registrar.Channel, b.Publisher.Connect())

	// Init and Start spooler: Harvesters dump events into the spooler.
	spooler, err := spooler.New(config, publisherChan)
	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	crawler, err := crawler.New(spooler, config.Prospectors)
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

	// Starting spooler
	spooler.Start()
	// Stopping spooler will flush items
	defer spooler.Stop()

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

// Cleanup removes any temporary files, data, or other items that were created by the Beat.
func (fb *Filebeat) Cleanup(b *beat.Beat) error {
	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (fb *Filebeat) Stop() {

	logp.Info("Stopping filebeat")

	// Stop Filebeat
	close(fb.done)
}
