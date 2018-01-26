package tcp

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := input.Register("tcp", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input for TCP connection
type Input struct {
	harvester *Harvester
	started   bool
	outlet    channel.Outleter
}

// NewInput creates a new TCP input
func NewInput(cfg *common.Config, outlet channel.Factory, context input.Context) (input.Input, error) {
	cfgwarn.Experimental("TCP input type is used")

	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)
	harvester, err := NewHarvester(forwarder, cfg)
	if err != nil {
		return nil, err
	}
	return &Input{
		outlet:    out,
		harvester: harvester,
		started:   false,
	}, nil
}

// Run start a TCP input
func (p *Input) Run() {
	if !p.started {
		logp.Info("Starting TCP input")
		p.started = true

		go func() {
			defer p.outlet.Close()
			err := p.harvester.Run()
			if err != nil {
				logp.Err("Error running TCP harvester, error: %s", err)
			}
		}()
	}
}

// Stop stops TCP server
func (p *Input) Stop() {
	logp.Info("Stopping TCP input")
	p.harvester.Stop()
}

// Wait stop the current harvester
func (p *Input) Wait() {
	p.Stop()
}
