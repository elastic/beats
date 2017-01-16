package prospector

import (
	"fmt"
	"sync/atomic"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

type ProspectorStdin struct {
	harvester  *harvester.Harvester
	Prospector *Prospector
	started    bool
}

// NewProspectorStdin creates a new stdin prospector
// This prospector contains one harvester which is reading from stdin
func NewProspectorStdin(p *Prospector) (*ProspectorStdin, error) {

	prospectorer := &ProspectorStdin{}
	prospectorer.Prospector = p

	var err error

	prospectorer.harvester, err = p.createHarvester(file.State{Source: "-"})
	if err != nil {
		return nil, fmt.Errorf("Error initializing stdin harvester: %v", err)
	}

	return prospectorer, nil
}

func (p *ProspectorStdin) Init(states []file.State) error {
	p.started = false
	return nil
}

func (p *ProspectorStdin) Run() {

	// Make sure stdin harvester is only started once
	if !p.started {
		reader, err := p.harvester.Setup()
		if err != nil {
			logp.Err("Error starting stdin harvester: %s", err)
			return
		}

		p.Prospector.wg.Add(1)
		atomic.AddUint64(&p.Prospector.harvesterCounter, 1)

		go func() {
			defer func() {
				atomic.AddUint64(&p.Prospector.harvesterCounter, ^uint64(0))
				p.Prospector.wg.Done()
			}()

			p.harvester.Harvest(reader)
			p.started = true
		}()
	}
}
