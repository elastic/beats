package stdin

import (
	"fmt"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector/log"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

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
func NewProspector(cfg *common.Config, outlet channel.Outleter) (*Prospector, error) {

	p := &Prospector{
		started:  false,
		cfg:      cfg,
		outlet:   outlet,
		registry: harvester.NewRegistry(),
	}

	var err error

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
		p.registry.Start(p.harvester)
		p.started = true
	}
}

// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state file.State) (*log.Harvester, error) {

	// Each harvester gets its own copy of the outlet
	outlet := p.outlet.Copy()
	h, err := log.NewHarvester(
		p.cfg,
		state,
		nil,
		outlet,
	)

	return h, err
}

// Wait waits for completion of the prospector.
func (p *Prospector) Wait() {}

// Stop stops the prospector.
func (p *Prospector) Stop() {

}
