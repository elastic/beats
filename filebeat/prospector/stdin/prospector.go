package stdin

import (
	"fmt"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/prospector/log"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := prospector.Register("stdin", NewProspector)
	if err != nil {
		panic(err)
	}
}

// Prospector is a prospector for stdin
type Prospector struct {
	harvester *log.Harvester
	started   bool
	cfg       *common.Config
	outlet    channel.Outleter
	registry  *harvester.Registry
}

// NewStdin creates a new stdin prospector
// This prospector contains one harvester which is reading from stdin
func NewProspector(cfg *common.Config, outlet channel.Factory, context prospector.Context) (prospector.Prospectorer, error) {
	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	p := &Prospector{
		started:  false,
		cfg:      cfg,
		outlet:   out,
		registry: harvester.NewRegistry(),
	}

	p.harvester, err = p.createHarvester(file.State{Source: "-"})
	if err != nil {
		return nil, fmt.Errorf("Error initializing stdin harvester: %v", err)
	}

	return p, nil
}

// Run runs the prospector
func (p *Prospector) Run() {
	// Make sure stdin harvester is only started once
	if !p.started {
		err := p.harvester.Setup()
		if err != nil {
			logp.Err("Error setting up stdin harvester: %s", err)
			return
		}
		if err = p.registry.Start(p.harvester); err != nil {
			logp.Err("Error starting the harvester: %s", err)
		}
		p.started = true
	}
}

// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state file.State) (*log.Harvester, error) {
	// Each harvester gets its own copy of the outlet
	h, err := log.NewHarvester(
		p.cfg,
		state, nil, nil,
		p.outlet,
	)

	return h, err
}

// Wait waits for completion of the prospector.
func (p *Prospector) Wait() {}

// Stop stops the prospector.
func (p *Prospector) Stop() {
	p.outlet.Close()
}
