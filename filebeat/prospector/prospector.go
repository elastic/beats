package prospector

import (
	"errors"
	"expvar"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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
	cfg              *common.Config // Raw config
	config           prospectorConfig
	prospectorer     Prospectorer
	outlet           Outlet
	harvesterChan    chan *input.Event
	done             chan struct{}
	states           *file.States
	wg               sync.WaitGroup
	channelWg        sync.WaitGroup // Separate waitgroup for channels as not stopped on completion
	harvesterCounter uint64
}

type Prospectorer interface {
	Init(states file.States) error
	Run()
}

type Outlet interface {
	OnEvent(event *input.Event) bool
}

func NewProspector(cfg *common.Config, states file.States, outlet Outlet) (*Prospector, error) {
	prospector := &Prospector{
		cfg:           cfg,
		config:        defaultConfig,
		outlet:        outlet,
		harvesterChan: make(chan *input.Event),
		done:          make(chan struct{}),
		wg:            sync.WaitGroup{},
		states:        &file.States{},
		channelWg:     sync.WaitGroup{},
	}

	if err := cfg.Unpack(&prospector.config); err != nil {
		return nil, err
	}
	if err := prospector.config.Validate(); err != nil {
		return nil, err
	}

	err := prospector.Init(states)
	if err != nil {
		return nil, err
	}

	logp.Debug("prospector", "File Configs: %v", prospector.config.Paths)

	return prospector, nil
}

// Init sets up default config for prospector
func (p *Prospector) Init(states file.States) error {

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

	err = prospectorer.Init(states)
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

// Starts scanning through all the file paths and fetch the related files. Start a harvester for each file
func (p *Prospector) Run(once bool) {

	logp.Info("Starting prospector of type: %v", p.config.InputType)

	// This waitgroup is not needed if run only once
	// Waitgroup has to be added here to prevent panic in case Stop is called immediately afterwards
	if !once {
		// Add waitgroup to make sure prospectors finished
		p.wg.Add(1)
		defer p.wg.Done()
	}
	// Open channel to receive events from harvester and forward them to spooler
	// Here potential filtering can happen
	p.channelWg.Add(1)
	go func() {
		defer p.channelWg.Done()
		for {
			select {
			case <-p.done:
				logp.Info("Prospector channel stopped")
				return
			case event := <-p.harvesterChan:
				err := p.updateState(event)
				if err != nil {
					return
				}
			}
		}
	}()

	// Initial prospector run
	p.prospectorer.Run()

	// Shuts down after the first complete scan of all prospectors
	// As all harvesters are part of the prospector waitgroup, this waits for the closing of all harvesters
	if once {
		p.wg.Wait()
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

func (p *Prospector) Stop() {
	logp.Info("Stopping Prospector")
	close(p.done)
	p.channelWg.Wait()
	p.wg.Wait()
}

// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state file.State) (*harvester.Harvester, error) {

	h, err := harvester.NewHarvester(
		p.cfg,
		state,
		p.harvesterChan,
		p.done,
	)

	return h, err
}

// startHarvester starts a new harvester with the given offset
// In case the HarvesterLimit is reached, an error is returned
func (p *Prospector) startHarvester(state file.State, offset int64) error {

	if p.config.HarvesterLimit > 0 && atomic.LoadUint64(&p.harvesterCounter) >= p.config.HarvesterLimit {
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

	reader, err := h.Setup()
	if err != nil {
		return fmt.Errorf("Error setting up harvester: %s", err)
	}

	// State is directly updated and not through channel to make state update immediate
	// State is only updated after setup is completed successfully
	err = p.updateState(input.NewEvent(state))
	if err != nil {
		return err
	}

	p.wg.Add(1)
	// startHarvester is not run concurrently, but atomic operations are need for the decrementing of the counter
	// inside the following go routine
	atomic.AddUint64(&p.harvesterCounter, 1)
	go func() {
		defer func() {
			atomic.AddUint64(&p.harvesterCounter, ^uint64(0))
			p.wg.Done()
		}()

		// Starts harvester and picks the right type. In case type is not set, set it to defeault (log)
		h.Harvest(reader)
	}()

	return nil
}
