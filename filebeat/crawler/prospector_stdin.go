package crawler

import (
	"fmt"
	"time"

	"github.com/elastic/beats/filebeat/harvester"
)

type ProspectorStdin struct {
	Prospector *Prospector
	harvester  *harvester.Harvester
	started    bool
}

func NewProspectorStdin(p *Prospector) (*ProspectorStdin, error) {

	prospectorer := &ProspectorStdin{
		Prospector: p,
	}

	var err error

	prospectorer.harvester, err = p.CreateHarvester("-", nil)

	if err != nil {
		return nil, fmt.Errorf("Error initializing stdin harvester: %v", err)
	}

	return prospectorer, nil
}

func (p *ProspectorStdin) Init() {
	p.started = false
}

func (p *ProspectorStdin) Run() {

	// Make sure stdin harvester is only started once
	if !p.started {
		p.Prospector.RunHarvester(p.harvester)
	}

	// Wait time during endless loop
	oneSecond, _ := time.ParseDuration("1s")
	time.Sleep(oneSecond)
}
