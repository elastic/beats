package udp

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Prospector struct {
	harvester *Harvester
	started   bool
}

func NewProspector(cfg *common.Config, outlet channel.Outleter) (*Prospector, error) {
	forwarder, err := harvester.NewForwarder(cfg, outlet)
	if err != nil {
		return nil, err
	}
	return &Prospector{
		harvester: NewHarvester(forwarder, cfg),
		started:   false,
	}, nil
}

func (p *Prospector) Run() {
	logp.Info("Starting udp prospector")

	if !p.started {
		p.started = true
		go func() {
			err := p.harvester.Run()
			if err != nil {
				logp.Err("Error running harvester:: %v", err)
			}
		}()
	}
}

func (p *Prospector) Stop() {
	logp.Info("Stopping udp prospector")
	p.harvester.Stop()
}

func (p *Prospector) Wait() {
	p.Stop()
}
