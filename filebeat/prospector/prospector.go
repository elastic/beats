package prospector

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector/stdin"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	harvesterSkipped = monitoring.NewInt(nil, "filebeat.harvester.skipped")
)

// Prospector contains the prospector
type Prospector struct {
	cfg          *common.Config // Raw config
	config       prospectorConfig
	prospectorer Prospectorer
	outlet       channel.Outleter
	done         chan struct{}
	states       *file.States
	wg           *sync.WaitGroup
	id           uint64
	Once         bool
	registry     *harvesterRegistry
	beatDone     chan struct{}
}

// Prospectorer is the interface common to all prospectors
type Prospectorer interface {
	LoadStates(states []file.State) error
	Run()
}

// NewProspector instantiates a new prospector
func NewProspector(cfg *common.Config, outlet channel.Outleter, beatDone chan struct{}) (*Prospector, error) {
	prospector := &Prospector{
		cfg:      cfg,
		config:   defaultConfig,
		outlet:   outlet,
		wg:       &sync.WaitGroup{},
		done:     make(chan struct{}),
		states:   &file.States{},
		Once:     false,
		registry: newHarvesterRegistry(),
		beatDone: beatDone,
	}

	var err error
	if err = cfg.Unpack(&prospector.config); err != nil {
		return nil, err
	}

	var h map[string]interface{}
	cfg.Unpack(&h)
	prospector.id, err = hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	logp.Debug("prospector", "File Configs: %v", prospector.config.Paths)

	return prospector, nil
}

// LoadStates sets up default config for prospector
func (p *Prospector) LoadStates(states []file.State) error {

	var prospectorer Prospectorer
	var err error

	switch p.config.InputType {
	case cfg.StdinInputType:
		prospectorer, err = stdin.NewProspector(p.cfg, p.outlet)
	case cfg.LogInputType:
		prospectorer, err = NewLog(p)
	default:
		return fmt.Errorf("Invalid input type: %v", p.config.InputType)
	}

	if err != nil {
		return err
	}

	err = prospectorer.LoadStates(states)
	if err != nil {
		return err
	}
	p.prospectorer = prospectorer

	// Create empty harvester to check if configs are fine
	_, err = p.createHarvester(file.State{})
	if err != nil {
		return err
	}

	return nil
}

// Start starts the prospector
func (p *Prospector) Start() {
	p.wg.Add(1)
	logp.Info("Starting prospector of type: %v; id: %v ", p.config.InputType, p.ID())

	onceWg := sync.WaitGroup{}
	if p.Once {
		// Make sure start is only completed when Run did a complete first scan
		defer onceWg.Wait()
	}

	onceWg.Add(1)
	// Add waitgroup to make sure prospectors finished
	go func() {
		defer func() {
			onceWg.Done()
			p.stop()
			p.wg.Done()
		}()

		p.Run()
	}()

}

// Run starts scanning through all the file paths and fetch the related files. Start a harvester for each file
func (p *Prospector) Run() {

	// Initial prospector run
	p.prospectorer.Run()

	// Shuts down after the first complete run of all prospectors
	if p.Once {
		return
	}

	for {
		select {
		case <-p.done:
			logp.Info("Prospector ticker stopped")
			return
		case <-time.After(p.config.ScanFrequency):
			logp.Debug("prospector", "Run prospector")
			p.prospectorer.Run()
		}
	}
}

// ID returns prospector identifier
func (p *Prospector) ID() uint64 {
	return p.id
}

// updateState updates the prospector state and forwards the event to the spooler
// All state updates done by the prospector itself are synchronous to make sure not states are overwritten
func (p *Prospector) updateState(state file.State) error {

	// Add ttl if cleanOlder is enabled and TTL is not already 0
	if p.config.CleanInactive > 0 && state.TTL != 0 {
		state.TTL = p.config.CleanInactive
	}

	// Update first internal state
	p.states.Update(state)

	data := util.NewData()
	data.SetState(state)

	ok := p.outlet.OnEvent(data)

	if !ok {
		logp.Info("Prospector outlet closed")
		return errors.New("prospector outlet closed")
	}

	return nil
}

// Stop stops the prospector and with it all harvesters
func (p *Prospector) Stop() {
	// Stop scanning and wait for completion
	close(p.done)
	p.wg.Wait()
}

func (p *Prospector) stop() {
	logp.Info("Stopping Prospector: %v", p.ID())

	// In case of once, it will be waited until harvesters close itself
	if p.Once {
		p.registry.waitForCompletion()
	}

	// Stop all harvesters
	// In case the beatDone channel is closed, this will not wait for completion
	// Otherwise Stop will wait until output is complete
	p.registry.Stop()
}

// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state file.State) (*harvester.Harvester, error) {

	// Each harvester gets its own copy of the outlet
	outlet := p.outlet.Copy()
	h, err := harvester.NewHarvester(
		p.cfg,
		state,
		p.states,
		outlet,
	)

	return h, err
}

// startHarvester starts a new harvester with the given offset
// In case the HarvesterLimit is reached, an error is returned
func (p *Prospector) startHarvester(state file.State, offset int64) error {

	if p.config.HarvesterLimit > 0 && p.registry.len() >= p.config.HarvesterLimit {
		harvesterSkipped.Add(1)
		return fmt.Errorf("Harvester limit reached")
	}

	// Set state to "not" finished to indicate that a harvester is running
	state.Finished = false
	state.Offset = offset

	// Create harvester with state
	h, err := p.createHarvester(state)
	if err != nil {
		return err
	}

	reader, err := h.Setup()
	if err != nil {
		return fmt.Errorf("Error setting up harvester: %s", err)
	}

	p.registry.start(h, reader)

	return nil
}
