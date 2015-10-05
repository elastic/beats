package beat

import (
	"fmt"
	"os"
	"time"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"

	cfg "github.com/elastic/filebeat/config"
	. "github.com/elastic/filebeat/crawler"
	. "github.com/elastic/filebeat/input"
)

// TODO: Cleanup if possible
var exitStat = struct {
	ok, usageError, faulted int
}{
	ok:         0,
	usageError: 1,
	faulted:    2,
}

// Beater object. Contains all objects needed to run the beat
type Filebeat struct {
	FbConfig *cfg.Config
	// Channel from harvesters to spooler
	SpoolChan     chan *FileEvent
	publisherChan chan []*FileEvent
	RegistrarChan chan []*FileEvent
	Spooler       *Spooler
}

// Config setups up the filebeat configuration by fetch all additional config files
func (fb *Filebeat) Config(b *beat.Beat) error {

	// Load Base config
	err := cfgfile.Read(&fb.FbConfig, "")

	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	// Check if optional configDir is set to fetch additional prospector config files
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
		os.Exit(exitStat.faulted)
	}()

	// Init channels
	fb.SpoolChan = make(chan *FileEvent, 16)
	fb.publisherChan = make(chan []*FileEvent, 1)
	fb.RegistrarChan = make(chan []*FileEvent, 1)

	// Setup registrar to persist state
	registrar := NewRegistrar(fb.FbConfig.Filebeat.RegistryFile)

	crawl := &Crawler{
		Registrar: registrar,
	}

	// Load the previous log file locations now, for use in prospector
	registrar.LoadState()
	crawl.Start(fb.FbConfig.Filebeat.Prospectors, fb.SpoolChan)

	// Init and Start spooler: Harvesters dump events into the spooler.
	spooler := NewSpooler(fb)
	err := spooler.Config()

	if err != nil {
		logp.Err("Could not init spooler: %v", err)
		return err
	}

	fb.Spooler = spooler

	// TODO: Check if spooler shouldn't start earlier?
	go spooler.Start()

	// Publishes event to output
	go Publish(b, fb)

	// registrar records last acknowledged positions in all files.
	registrar.WriteState(fb.RegistrarChan)

	return nil
}

func (fb *Filebeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (fb *Filebeat) Stop() {

	// Stop harvesters
	// Stop prospectors
	// Flush what is in spooler
	// Write state

	fb.Spooler.Stop()

	// FIXME: Improve to first write state and then close channels
	close(fb.SpoolChan)
	close(fb.publisherChan)
	close(fb.RegistrarChan)
}

func Publish(beat *beat.Beat, fb *Filebeat) {

	// Receives events from spool during flush
	for events := range fb.publisherChan {

		pubEvents := make([]common.MapStr, 0, len(events))

		logp.Debug("filebeat", "Send events to output")
		for _, event := range events {
			bEvent := common.MapStr{
				"timestamp": common.Time(time.Now()),
				"source":    event.Source,
				"offset":    event.Offset,
				"message":   event.Line,
				"text":      event.Text,
				"fields":    event.Fields,
				"fileinfo":  event.Fileinfo,
				"type":      "log",
			}

			pubEvents = append(pubEvents, bEvent)
		}

		publishEvents(beat.Events, pubEvents)

		logp.Debug("filebeat", "Events sent: %d", len(events))

		// Tell the registrar that we've successfully sent these events
		fb.RegistrarChan <- events
	}
}

func publishEvents(client publisher.Client, events []common.MapStr) {

	// Sends event to beat (outputs).
	// Wait/Repeat until all events are published
	for {
		ok := client.PublishEvents(events, publisher.Confirm)
		if ok {
			break
		}
	}
}
