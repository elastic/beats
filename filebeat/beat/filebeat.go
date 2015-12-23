package beat

import (
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	cfg "github.com/elastic/beats/filebeat/config"
	. "github.com/elastic/beats/filebeat/crawler"
	. "github.com/elastic/beats/filebeat/input"
)

// Beater object. Contains all objects needed to run the beat
type Filebeat struct {
	FbConfig *cfg.Config
	// Channel from harvesters to spooler
	publisherChan chan []*FileEvent
	Spooler       *Spooler
	registrar     *Registrar
}

func New() *Filebeat {
	return &Filebeat{}
}

// Config setups up the filebeat configuration by fetch all additional config files
func (fb *Filebeat) Config(b *beat.Beat) error {

	// Load Base config
	err := cfgfile.Read(&fb.FbConfig, "")

	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	// Check if optional config_dir is set to fetch additional prospector config files
	fb.FbConfig.FetchConfigs()

	return nil
}

func (fb *Filebeat) Setup(b *beat.Beat) error {
	return nil
}

func (fb *Filebeat) Run(b *beat.Beat) error {

	defer func() {
		p := recover()
		if p == nil {
			return
		}

		fmt.Printf("recovered panic: %v", p)
		os.Exit(1)
	}()

	var err error

	// Init channels
	fb.publisherChan = make(chan []*FileEvent, 1)

	// Setup registrar to persist state
	fb.registrar, err = NewRegistrar(fb.FbConfig.Filebeat.RegistryFile)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	crawl := &Crawler{
		Registrar: fb.registrar,
	}

	// Load the previous log file locations now, for use in prospector
	fb.registrar.LoadState()

	// Init and Start spooler: Harvesters dump events into the spooler.
	fb.Spooler = NewSpooler(fb)
	err = fb.Spooler.Config()

	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	// Start up spooler
	go fb.Spooler.Run()

	crawl.Start(fb.FbConfig.Filebeat.Prospectors, fb.Spooler.Channel)

	// Publishes event to output
	go Publish(b, fb)

	// registrar records last acknowledged positions in all files.
	fb.registrar.Run()

	return nil
}

func (fb *Filebeat) Cleanup(b *beat.Beat) error {
	return nil
}

// Stop is called on exit for cleanup
func (fb *Filebeat) Stop() {

	// Stop harvesters
	// Stop prospectors

	// Stopping spooler will flush items
	fb.Spooler.Stop()

	// Stopping registrar will write last state
	fb.registrar.Stop()

	// Close channels
	//close(fb.publisherChan)
}

func Publish(beat *beat.Beat, fb *Filebeat) {
	logp.Info("Start sending events to output")

	// Receives events from spool during flush
	for events := range fb.publisherChan {

		pubEvents := make([]common.MapStr, 0, len(events))
		for _, event := range events {
			pubEvents = append(pubEvents, event.ToMapStr())
		}

		beat.Events.PublishEvents(pubEvents, publisher.Sync, publisher.Guaranteed)

		logp.Info("Events sent: %d", len(events))

		// Tell the registrar that we've successfully sent these events
		fb.registrar.Channel <- events
	}
}
