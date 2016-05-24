package crawler

import (
	"fmt"
	"sync"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Prospector struct {
	config        prospectorConfig
	prospectorer  Prospectorer
	spoolerChan   chan *input.FileEvent
	harvesterChan chan *input.FileEvent
	done          chan struct{}
	states        *input.States
	wg            sync.WaitGroup
}

type Prospectorer interface {
	Init()
	Run()
}

func NewProspector(cfg *common.Config, states input.States, spoolerChan chan *input.FileEvent) (*Prospector, error) {
	prospector := &Prospector{
		config:        defaultConfig,
		spoolerChan:   spoolerChan,
		harvesterChan: make(chan *input.FileEvent),
		done:          make(chan struct{}),
		states:        states.Copy(),
		wg:            sync.WaitGroup{},
	}

	if err := cfg.Unpack(&prospector.config); err != nil {
		return nil, err
	}

	err := prospector.Init()
	if err != nil {
		return nil, err
	}

	logp.Debug("prospector", "File Configs: %v", prospector.config.Paths)

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

	switch p.config.Harvester.InputType {
	case cfg.StdinInputType:
		prospectorer, err = NewProspectorStdin(p)
	case cfg.LogInputType:
		prospectorer, err = NewProspectorLog(p)
	default:
		return fmt.Errorf("Invalid prospector type: %v", p.config.Harvester.InputType)
	}

	prospectorer.Init()
	p.prospectorer = prospectorer

	return nil
}

// Starts scanning through all the file paths and fetch the related files. Start a harvester for each file
func (p *Prospector) Run() {

	logp.Info("Starting prospector of type: %v", p.config.Harvester.InputType)
	p.wg.Add(2)
	defer p.wg.Done()

	// Open channel to receive events from harvester and forward them to spooler
	// Here potential filtering can happen
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.done:
				logp.Info("Prospector channel stopped")
				return
			case event := <-p.harvesterChan:
				select {
				case <-p.done:
					logp.Info("Prospector channel stopped")
					return
				case p.spoolerChan <- event:
					p.states.Update(event.FileState)
				}
			}
		}
	}()

	// Initial prospector run
	p.prospectorer.Run()

	for {
		select {
		case <-p.done:
			logp.Info("Prospector ticker stopped")
			return
		case <-time.After(p.config.ScanFrequency):
			logp.Info("Run prospector")
			p.prospectorer.Run()
		}
	}
}

func (p *Prospector) Stop(wg *sync.WaitGroup) {
	logp.Info("Stopping Prospector")
	close(p.done)
	p.wg.Wait()
	wg.Done()
}

// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state input.FileState) (*harvester.Harvester, error) {

	h, err := harvester.NewHarvester(
		&p.config.Harvester,
		state.Source,
		state,
		p.harvesterChan,
		state.Offset,
		p.done,
	)

	return h, err
}

func (p *Prospector) startHarvester(state input.FileState, offset int64) (*harvester.Harvester, error) {
	state.Offset = offset
	// Create harvester with state
	h, err := p.createHarvester(state)
	if err != nil {
		return nil, err
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		// Starts harvester and picks the right type. In case type is not set, set it to defeault (log)
		h.Harvest()
	}()

	return h, nil
}

// Setup Prospector Config
func (p *Prospector) setupProspectorConfig() error {
	config := &p.config

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
	config := &p.config.Harvester

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

	config.CloseOlderDuration, err = getConfigDuration(config.CloseOlder, cfg.DefaultCloseOlder, "close_older")
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

// isIgnoreOlder checks if the given state reached ignore_older
func (p *Prospector) isIgnoreOlder(state input.FileState) bool {

	// ignore_older is disable
	if p.config.IgnoreOlder == 0 {
		return false
	}

	modTime := state.Fileinfo.ModTime()

	if time.Since(modTime) > p.config.IgnoreOlder {
		return true
	}

	return false
}
