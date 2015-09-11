package beat

import (
	"flag"
	"fmt"
	cfg "github.com/elastic/filebeat/config"
	. "github.com/elastic/filebeat/crawler"
	. "github.com/elastic/filebeat/input"
	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"os"
	"time"
	"github.com/elastic/libbeat/cfgfile"
)

var exitStat = struct {
	ok, usageError, faulted int
}{
	ok:         0,
	usageError: 1,
	faulted:    2,
}

var configDirPath string

// Init config path flag
func init() {
	flag.StringVar(&configDirPath, "configDir", "", "path to additional filebeat configuration directory with .yml files")
}

// Beater object. Contains all objects needed to run the beat
type Filebeat struct {
	FbConfig *cfg.Config
	// Channel from harvesters to spooler
	SpoolChan     chan *FileEvent
	publisherChan chan []*FileEvent
	RegistrarChan chan []*FileEvent
}

// Config setups up the filebeat configuration by fetch all additional config files
func (fb *Filebeat) Config(b *beat.Beat) error {

	emitOptions()

	// Load Base config
	err := cfgfile.Read(&fb.FbConfig, "")

	if err != nil {
		logp.Warn("Error reading config file:", err)
	}

	// This is optiona
	if configDirPath != "" {
		logp.Info("Additional config files are fetched from:", configDirPath)
		fb.FbConfig.FetchConfigs(configDirPath)
	}

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
	persist := make(map[string]*FileState)

	restart := &ProspectorResume{}
	restart.LoadState()
	restart.Scan(fb.FbConfig.Filebeat.Files, persist, fb.SpoolChan)

	// Start spooler: Harvesters dump events into the spooler.
	go fb.startSpooler(cfg.CmdlineOptions)

	// Publishes event to output
	go Publish(b, fb)

	// registrar records last acknowledged positions in all files.
	Registrar(persist, fb.RegistrarChan)

	return nil
}

func (fb *Filebeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (fb *Filebeat) Stop() {

	// FIXME: Improve to first write state and then close channels
	close(fb.SpoolChan)
	close(fb.publisherChan)
	close(fb.RegistrarChan)
}

// emitOptions prints out the set config options
func emitOptions() {
	logp.Info("filebeat", "\t--- options -------")
	logp.Info("filebeat", "\tconfig-arg:          %s", configDirPath)
	logp.Info("filebeat", "\tidle-timeout:        %v", cfg.CmdlineOptions.IdleTimeout)
	logp.Info("filebeat", "\tspool-size:          %d", cfg.CmdlineOptions.SpoolSize)
	logp.Info("filebeat", "\tharvester-buff-size: %d", cfg.CmdlineOptions.HarvesterBufferSize)
	logp.Info("filebeat", "\t--- flags ---------")
	logp.Info("filebeat", "\ttail (on-rotation):  %t", cfg.CmdlineOptions.TailOnRotate)
	logp.Info("filebeat", "\tquiet:             %t", cfg.CmdlineOptions.Quiet)
}

func Publish(beat *beat.Beat, fb *Filebeat) {

	// Receives events from spool during flush
	for events := range fb.publisherChan {

		logp.Debug("filebeat", "Send events to output")
		for _, event := range events {

			bEvent := common.MapStr{
				"timestamp": common.Time(time.Now()),
				"source":    event.Source,
				"offset":    event.Offset,
				"line":      event.Line,
				"text":      event.Text,
				"fields":    event.Fields,
				"fileinfo":  event.Fileinfo,
				"type":      "log",
			}

			// Sends event to beat (outputs)
			beat.Events <- bEvent
		}

		logp.Debug("filebeat", "Events sent:", len(events))

		// Tell the registrar that we've successfully sent these events
		fb.RegistrarChan <- events
	}
}
