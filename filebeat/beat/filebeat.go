package beat

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/logp"

	cfg "github.com/elastic/beats/filebeat/config"
	. "github.com/elastic/beats/filebeat/crawler"
	. "github.com/elastic/beats/filebeat/input"
)

// Beater object. Contains all objects needed to run the beat
type Filebeat struct {
	FbConfig *cfg.Config
	// Channel from harvesters to spooler
	publisherChan chan []*FileEvent
	spooler       *Spooler
	registrar     *Registrar
	cralwer       *Crawler
	done          chan struct{}
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
	fb.done = make(chan struct{})

	return nil
}

func (fb *Filebeat) Run(b *beat.Beat) error {

	var err error

	// Init channels
	fb.publisherChan = make(chan []*FileEvent, 1)

	// Setup registrar to persist state
	fb.registrar, err = NewRegistrar(fb.FbConfig.Filebeat.RegistryFile)
	if err != nil {
		logp.Err("Could not init registrar: %v", err)
		return err
	}

	fb.cralwer = &Crawler{
		Registrar: fb.registrar,
	}

	// Load the previous log file locations now, for use in prospector
	fb.registrar.LoadState()

	// Init and Start spooler: Harvesters dump events into the spooler.
	fb.spooler = NewSpooler(fb.FbConfig.Filebeat, fb.publisherChan)

	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	// Start up spooler
	fb.spooler.Start()

	// registrar records last acknowledged positions in all files.
	go fb.registrar.Run()

	err = fb.cralwer.Start(fb.FbConfig.Filebeat.Prospectors, fb.spooler.Channel)
	if err != nil {
		return err
	}

	// Publishes event to output
	pub := newPublisher(fb.FbConfig.Filebeat.PublishAsync,
		fb.publisherChan, fb.registrar.Channel, b.Events)
	pub.Start()

	// Blocks progressing
	select {
	case <-fb.done:
	}

	return nil
}

func (fb *Filebeat) Cleanup(b *beat.Beat) error {
	return nil
}

// Stop is called on exit for cleanup
func (fb *Filebeat) Stop() {

	logp.Info("Stopping filebeat")
	// Stop crawler -> stop prospectors -> stop harvesters
	fb.cralwer.Stop()

	// Stopping spooler will flush items
	fb.spooler.Stop()

	// Stopping registrar will write last state
	fb.registrar.Stop()

	// Stop Filebeat
	close(fb.done)
}
