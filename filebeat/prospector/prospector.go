package prospector

import (
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
	harvesterCounter uint64
}

type Prospectorer interface {
	Init()
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
	_, err = p.createHarvester(file.State{})
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
				// Add ttl if cleanOlder is enabled
				if p.config.CleanInactive > 0 {
					event.State.TTL = p.config.CleanInactive
				}

				ok := p.outlet.OnEvent(event)
				if !ok {
					logp.Info("Prospector outlet closed")
					return
				}

				p.states.Update(event.State)
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
			logp.Debug("prospector", "Run prospector")
			p.prospectorer.Run()
		}
	}
}

func (p *Prospector) Stop() {
	logp.Info("Stopping Prospector")
	close(p.done)
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
	// Create harvester with state
	h, err := p.createHarvester(state)
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
		h.Harvest()
	}()

	return nil
}
