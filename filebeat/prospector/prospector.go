package prospector

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
	cfg           *common.Config // Raw config
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
		cfg:           cfg,
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
	if err := prospector.config.Validate(); err != nil {
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

	var prospectorer Prospectorer
	var err error

	switch p.config.InputType {
	case cfg.StdinInputType:
		prospectorer, err = NewProspectorStdin(p)
	case cfg.LogInputType:
		prospectorer, err = NewProspectorLog(p)
	default:
		return fmt.Errorf("Invalid input type: %v", p.config.InputType)
	}

	if err != nil {
		return err
	}

	prospectorer.Init()
	p.prospectorer = prospectorer

	// Create empty harvester to check if configs are fine
	_, err = p.createHarvester(input.FileState{})
	if err != nil {
		return err
	}

	return nil
}

// Starts scanning through all the file paths and fetch the related files. Start a harvester for each file
func (p *Prospector) Run() {

	logp.Info("Starting prospector of type: %v", p.config.InputType)
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
		p.cfg,
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
