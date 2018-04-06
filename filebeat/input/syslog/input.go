package main

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

//go:generate ragel -Z -G2 parser.rl -o parser.go

func init() {
	err := input.Register("syslog", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input define a syslog input
type Input struct {
	harvester *Harvester
	started   bool
	outlet    channel.Outleter
}

// NewInput creates a new syslog input
func NewInput(
	cfg *common.Config,
	outlet channel.Factory,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Experimental("Syslog input type is used")

	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)
	return &Input{
		outlet:    out,
		harvester: factory(config, forwarder),
		started:   false,
	}
}

// Run starts and execute the UDP server.
func (p *Input) Run() {
	if !p.started {
		logp.Info("Starting Syslog input")
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

// Stop stops the TCP input
func (p *Input) Stop() {
	logp.Info("stopping Syslog input")
	p.harvester.Stop()
}

// Wait suspends the TCP input
func (p *Input) Wait() {
	p.Stop()
}

func factory(forwarder *harvester.Forwarder, config *Config) *harvester.Harvester {
	if config.isUDP() {
		havester.NewHarvester(forwarder, config)
	}
}
