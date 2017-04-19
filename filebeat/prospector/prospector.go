package prospector

import (
	"errors"
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	harvesterSkipped = expvar.NewInt("filebeat.harvester.skipped")
)

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
	channelWg     *sync.WaitGroup // Separate waitgroup for channels as not stopped on completion
	id            uint64
	Once          bool
	registry      *harvesterRegistry
	beatDone      chan struct{}
	eventCounter  *sync.WaitGroup
}

type Prospectorer interface {
	LoadStates(states []file.State) error
	Run()
}

type Outlet interface {
	OnEvent(event *input.Event) bool
}

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
		channelWg:     &sync.WaitGroup{},
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

// Init sets up default config for prospector
func (p *Prospector) LoadStates(states []file.State) error {

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

func (p *Prospector) Start() {
	p.wg.Add(1)
	logp.Info("Starting prospector of type: %v; id: %v ", p.config.InputType, p.ID())

	// Open channel to receive events from harvester and forward them to spooler
	// Here potential filtering can happen
	p.channelWg.Add(1)
	go func() {
		defer p.channelWg.Done()
		for {
			select {
			case <-p.channelDone:
				logp.Info("Prospector channel stopped")
				return
			case <-p.beatDone:
				logp.Info("Prospector channel stopped because beat is stopping.")
				return
			case event := <-p.harvesterChan:
				// No stopping on error, because on error it is expected that beatDone is closed
				// in the next run. If not, this will further drain the channel.
				p.updateState(event)
				p.eventCounter.Done()
			}
		}
	}()

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

// Starts scanning through all the file paths and fetch the related files. Start a harvester for each file
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
func (p *Prospector) updateState(event *input.Event) error {

	// Add ttl if cleanOlder is enabled and TTL is not already 0
	if p.config.CleanInactive > 0 && event.State.TTL != 0 {
		event.State.TTL = p.config.CleanInactive
	}

	ok := p.outlet.OnEvent(event)
	if !ok {
		logp.Info("Prospector outlet closed")
		return errors.New("prospector outlet closed")
	}

	p.states.Update(event.State)
	return nil
}

// Stop stops the prospector and with it all harvesters
//
// The shutdown order is as follwoing
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
	// Waits until channel go-routine properly stopped
	p.channelWg.Wait()
}

// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state file.State) (*harvester.Harvester, error) {

	outlet := channel.NewOutlet(p.beatDone, p.harvesterChan, p.eventCounter)
	h, err := harvester.NewHarvester(
		p.cfg,
		state,
		outlet,
	)

	return h, err
}

// startHarvester starts a new harvester with the given offset
// In case the HarvesterLimit is reached, an error is returned
func (p *Prospector) startHarvester(state file.State, offset int64) error {

	if p.config.HarvesterLimit > 0 && p.registry.len() >= p.config.HarvesterLimit {
		harvesterSkipped.Add(1)
		return fmt.Errorf("Harvester limit reached.")
	}

	state.Offset = offset
	// Set state to "not" finished to indicate that a harvester is running
	state.Finished = false

	// Create harvester with state
	h, err := p.createHarvester(state)
	if err != nil {
		return err
	}

	// State is directly updated and not through channel to make state update synchronous
	err = p.updateState(input.NewEvent(state))
	if err != nil {
		return err
	}

	reader, err := h.Setup()
	if err != nil {
		// Set state to finished True again in case of setup failure to make sure
		// file can be picked up again by a future harvester
		state.Finished = true

		updateErr := p.updateState(input.NewEvent(state))
		// This should only happen in the case that filebeat is stopped
		if updateErr != nil {
			logp.Err("Error updating state: %v", updateErr)
		}
		return fmt.Errorf("Error setting up harvester: %s", err)
	}

	p.registry.start(h, reader)

	return nil
}
