package crawler

import (
	"fmt"
	"sync"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
)

type Prospector struct {
	ProspectorConfig cfg.ProspectorConfig
	prospectorer     Prospectorer
	channel          chan *input.FileEvent
	registrar        *Registrar
	done             chan struct{}
}

type Prospectorer interface {
	Init()
	Run()
}

func NewProspector(prospectorConfig cfg.ProspectorConfig, registrar *Registrar, channel chan *input.FileEvent) (*Prospector, error) {
	prospector := &Prospector{
		ProspectorConfig: prospectorConfig,
		registrar:        registrar,
		channel:          channel,
		done:             make(chan struct{}),
	}

	err := prospector.Init()

	if err != nil {
		return nil, err
	}

	return prospector, nil
}

// Init sets up default config for prospector
func (p *Prospector) Init() error {

	err := p.setupProspectorConfig()
	if err != nil {
		return err
	}

	err = p.setupHarvesterConfig()
	if err != nil {
		return err
	}

	var prospectorer Prospectorer

	switch p.ProspectorConfig.Harvester.InputType {
	case cfg.StdinInputType:
		prospectorer, err = NewProspectorStdin(p)
		prospectorer.Init()
	case cfg.LogInputType:
		prospectorer, err = NewProspectorLog(p)
		prospectorer.Init()

	default:
		return fmt.Errorf("Invalid prospector type: %v", p.ProspectorConfig.Harvester.InputType)
	}

	p.prospectorer = prospectorer

	return nil
}

// Starts scanning through all the file paths and fetch the related files. Start a harvester for each file
func (p *Prospector) Run(wg *sync.WaitGroup) {

	// TODO: Defer the wg.Done() call to block shutdown
	// Currently there are 2 cases where shutting down the prospector could be blocked:
	// 1. reading from file
	// 2. forwarding event to spooler
	// As this is not implemented yet, no blocking on prospector shutdown is done.
	wg.Done()

	logp.Info("Starting prospector of type: %v", p.ProspectorConfig.Harvester.InputType)

	for {
		select {
		case <-p.done:
			logp.Info("Prospector stopped")
			return
		default:
			logp.Info("Run prospector")
			p.prospectorer.Run()
		}
	}
}

func (p *Prospector) Stop() {
	logp.Info("Stopping Prospector")
	close(p.done)
}

func (p *Prospector) AddHarvester(file string, stat *harvester.FileStat) (*harvester.Harvester, error) {

	h, err := harvester.NewHarvester(
		&p.ProspectorConfig.Harvester, file, stat, p.channel)

	return h, err
}

// Setup Prospector Config
func (p *Prospector) setupProspectorConfig() error {
	var err error
	config := &p.ProspectorConfig

	config.IgnoreOlderDuration, err = getConfigDuration(config.IgnoreOlder, cfg.DefaultIgnoreOlderDuration, "ignore_older")
	if err != nil {
		return err
	}

	config.ScanFrequencyDuration, err = getConfigDuration(config.ScanFrequency, cfg.DefaultScanFrequency, "scan_frequency")
	if err != nil {
		return err
	}
	config.ExcludeFilesRegexp, err = harvester.InitRegexps(config.ExcludeFiles)
	if err != nil {
		return err
	}

	if config.Harvester.InputType == cfg.LogInputType && len(config.Paths) == 0 {
		return fmt.Errorf("No paths were defined for prospector")
	}

	if config.Harvester.JSON != nil && len(config.Harvester.JSON.MessageKey) == 0 &&
		config.Harvester.Multiline != nil {

		return fmt.Errorf("When using the JSON decoder and multiline together, you need to specify a message_key value")
	}

	if config.Harvester.JSON != nil && len(config.Harvester.JSON.MessageKey) == 0 &&
		(len(config.Harvester.IncludeLines) > 0 || len(config.Harvester.ExcludeLines) > 0) {

		return fmt.Errorf("When using the JSON decoder and line filtering together, you need to specify a message_key value")
	}

	return nil
}

// Setup Harvester Config
func (p *Prospector) setupHarvesterConfig() error {

	var err error
	config := &p.ProspectorConfig.Harvester

	// Setup Buffer Size
	if config.BufferSize <= 0 {
		config.BufferSize = cfg.DefaultHarvesterBufferSize
	}
	logp.Info("buffer_size set to: %v", config.BufferSize)

	// Setup DocumentType
	if config.DocumentType == "" {
		config.DocumentType = cfg.DefaultDocumentType
	}
	logp.Info("document_type set to: %v", config.DocumentType)

	// Setup InputType
	if _, ok := cfg.ValidInputType[config.InputType]; !ok {
		logp.Info("Invalid input type set: %v", config.InputType)
		config.InputType = cfg.DefaultInputType
	}
	logp.Info("input_type set to: %v", config.InputType)

	config.BackoffDuration, err = getConfigDuration(config.Backoff, cfg.DefaultBackoff, "backoff")
	if err != nil {
		return err
	}

	// Setup Backoff factor
	if config.BackoffFactor <= 0 {
		config.BackoffFactor = cfg.DefaultBackoffFactor
	}
	logp.Info("backoff_factor set to: %v", config.BackoffFactor)

	config.MaxBackoffDuration, err = getConfigDuration(config.MaxBackoff, cfg.DefaultMaxBackoff, "max_backoff")
	if err != nil {
		return err
	}

	if config.ForceCloseFiles {
		logp.Info("force_close_file is enabled")
	} else {
		logp.Info("force_close_file is disabled")
	}

	config.CloseOlderDuration, err = getConfigDuration(config.CloseOlder, cfg.DefaultCloseOlderDuration, "close_older")
	if err != nil {
		return err
	}

	if config.MaxBytes <= 0 {
		config.MaxBytes = cfg.DefaultMaxBytes
	}
	logp.Info("max_bytes set to: %v", config.MaxBytes)

	return nil
}

// getConfigDuration builds the duration based on the input string.
// Returns error if an invalid string duration is passed
// In case no duration is set, default duration will be used.
func getConfigDuration(config string, duration time.Duration, name string) (time.Duration, error) {

	// Setup Ignore Older
	if config != "" {
		var err error
		duration, err = time.ParseDuration(config)
		if err != nil {
			logp.Warn("Failed to parse %s value '%s'. Error was: %s\n", name, config)
			return 0, err
		}
	}
	logp.Info("Set %s duration to %s", name, duration)

	return duration, nil
}
