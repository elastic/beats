package udp

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := input.Register("udp", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input define a udp input
type Input struct {
	harvester *Harvester
	started   bool
	outlet    channel.Outleter
}

// NewInput creates a new udp input
func NewInput(
	cfg *common.Config,
	outlet channel.Factory,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Experimental("UDP input type is used")

	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)
	return &Input{
		outlet:    out,
		harvester: NewHarvester(forwarder, cfg),
		started:   false,
	}, nil
}

// Run starts and execute the UDP server.
func (p *Input) Run() {
	if !p.started {
		logp.Info("Starting udp input")
		p.started = true
		go func() {
			defer p.outlet.Close()
			err := p.harvester.Run()
			if err != nil {
				logp.Err("Error running harvester:: %v", err)
			}
		}()
	}
}

// Stop stops the UDP input
func (p *Input) Stop() {
	logp.Info("stopping UDP input")
	p.harvester.Stop()
}

// Wait suspends the UDP input
func (p *Input) Wait() {
	p.Stop()
}
