package udp

import (
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/filebeat/inputsource/udp"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
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

// Input defines a udp input to receive event on a specific host:port.
type Input struct {
	sync.Mutex
	udp     *udp.Server
	started bool
	outlet  channel.Outleter
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

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)
	callback := func(data []byte, metadata inputsource.NetworkMetadata) {
		e := util.NewData()
		e.Event = beat.Event{
			Timestamp: time.Now(),
			Meta: common.MapStr{
				"truncated": metadata.Truncated,
			},
			Fields: common.MapStr{
				"message": string(data),
				"source":  metadata.RemoteAddr.String(),
			},
		}
		forwarder.Send(e)
	}

	udp := udp.New(&config.Config, callback)

	return &Input{
		outlet:  out,
		udp:     udp,
		started: false,
	}, nil
}

// Run starts and start the UDP server and read events from the socket
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		logp.Info("Starting UDP input")
		err := p.udp.Start()
		if err != nil {
			logp.Err("Error running harvester: %v", err)
		}
		p.started = true
	}
}

// Stop stops the UDP input
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	logp.Info("Stopping UDP input")
	p.udp.Stop()
	p.started = false
}

// Wait suspends the UDP input
func (p *Input) Wait() {
	p.Stop()
}
