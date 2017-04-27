package prospector

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	harvesterSkipped = monitoring.NewInt(nil, "filebeat.harvester.skipped")
)

// Prospector contains the prospector
type Prospector struct {
	cfg           *common.Config // Raw config
	config        prospectorConfig
	prospectorer  Prospectorer
	outlet        Outlet
	harvesterChan chan *input.Event
	channelDone   chan struct{}
	runDone       chan struct{}
	runWg         *sync.WaitGroup
	states        *file.States
	wg            *sync.WaitGroup
	id            uint64
	Once          bool
	registry      *harvesterRegistry
	beatDone      chan struct{}
	eventCounter  *sync.WaitGroup
}

// Prospectorer is the interface common to all prospectors
type Prospectorer interface {
	LoadStates(states []file.State) error
	Run()
}

// Outlet is the outlet for a prospector
type Outlet interface {
	SetSignal(signal <-chan struct{})
	OnEventSignal(event *input.Data) bool
	OnEvent(event *input.Data) bool
	Copy() Outlet
}

// NewProspector instantiates a new prospector
func NewProspector(cfg *common.Config, outlet Outlet, beatDone chan struct{}) (*Prospector, error) {
	prospector := &Prospector{
		cfg:           cfg,
		config:        defaultConfig,
		outlet:        outlet,
		harvesterChan: make(chan *input.Event),
		channelDone:   make(chan struct{}),
		wg:            &sync.WaitGroup{},
		runDone:       make(chan struct{}),
		runWg:         &sync.WaitGroup{},
		states:        &file.States{},
		Once:          false,
		registry:      newHarvesterRegistry(),
		beatDone:      beatDone,
		eventCounter:  &sync.WaitGroup{},
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
		prospectorer, err = NewStdin(p)
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

	if p.Once {
		// Makes sure prospectors can complete first scan before stopped
		defer p.runWg.Wait()
	}

	// Add waitgroup to make sure prospectors finished
	p.runWg.Add(1)
	go func() {
		defer func() {
			p.runWg.Done()
			p.stop()
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
		case <-p.runDone:
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

	eventHolder := input.NewEvent(state).GetData()
	// Set to 0 as these are state updates only
	eventHolder.Metadata.Bytes = 0

	ok := p.outlet.OnEvent(&eventHolder)

	if !ok {
		logp.Info("Prospector outlet closed")
		return errors.New("prospector outlet closed")
	}

	return nil
}

// Stop stops the prospector and with it all harvesters
//
// The shutdown order is as following
// - stop run and scanning
// - wait until last scan finishes to make sure no new harvesters are added
// - stop harvesters
// - wait until all harvester finished
// - stop communication channel
// - wait on internal waitgroup to make sure all prospector go routines are stopped
// - wait until all events are forwarded to the spooler
func (p *Prospector) Stop() {
	// Stop scanning and wait for completion
	close(p.runDone)
	p.wg.Wait()
}

func (p *Prospector) stop() {
	defer p.wg.Done()

	logp.Info("Stopping Prospector: %v", p.ID())

	// In case of once, it will be waited until harvesters close itself
	if p.Once {
		p.registry.waitForCompletion()
	}

	// Wait for finishing of the running prospectors
	// This ensure no new harvesters are added.
	p.runWg.Wait()

	// Stop all harvesters
	// In case the beatDone channel is closed, this will not wait for completion
	// Otherwise Stop will wait until output is complete
	p.registry.Stop()

	// Waits on stopping all harvesters to make sure all events made it into the channel
	p.waitEvents()
}

// Wait for completion of sending events
func (p *Prospector) waitEvents() {

	done := make(chan struct{})
	go func() {
		p.eventCounter.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(p.channelDone)
	case <-p.beatDone:
	}
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
