package prospector

import (
	"fmt"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

// Stdin is a prospector for stdin
type Stdin struct {
	harvester *harvester.Harvester
	started   bool
}

// NewStdin creates a new stdin prospector
// This prospector contains one harvester which is reading from stdin
func NewStdin(p *Prospector) (*Stdin, error) {

	prospectorer := &Stdin{
		started: false,
	}

	var err error

	prospectorer.harvester, err = p.createHarvester(file.State{Source: "-"})
	if err != nil {
		return nil, fmt.Errorf("Error initializing stdin harvester: %v", err)
	}

	return prospectorer, nil
}

// LoadStates loads the states
func (s *Stdin) LoadStates(states []file.State) error {
	return nil
}

// Run runs the prospector
func (s *Stdin) Run() {

	// Make sure stdin harvester is only started once
	if !s.started {
		reader, err := s.harvester.Setup()
		if err != nil {
			logp.Err("Error starting stdin harvester: %s", err)
			return
		}
		go s.harvester.Harvest(reader)
		s.started = true
	}
}
