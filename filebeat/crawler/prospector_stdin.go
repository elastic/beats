package crawler

import (
	"fmt"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
)

type ProspectorStdin struct {
	Prospector *Prospector
	harvester  *harvester.Harvester
	started    bool
}

func NewProspectorStdin(config cfg.ProspectorConfig, channel chan *input.FileEvent) (*ProspectorStdin, error) {

	prospectorer := &ProspectorStdin{}

	var err error
	prospectorer.harvester, err = harvester.NewHarvester(
		config,
		&config.Harvester,
		"-",
		nil,
		channel,
	)

	if err != nil {
		return nil, fmt.Errorf("Error initializing stdin harvester: %v", err)
	}

	return prospectorer, nil
}

func (p ProspectorStdin) Init() {
	p.started = false
}

func (prospector ProspectorStdin) Run() {

	// Make sure stdin harvester is only started once
	if !prospector.started {
		prospector.harvester.Start()
	}

	// Wait time during endless loop
	oneSecond, _ := time.ParseDuration("1s")
	time.Sleep(oneSecond)

}
