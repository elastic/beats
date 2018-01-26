package tcp

import (
	"time"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/inputsource/tcp"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
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
	server  *tcp.Server
	started atomic.Bool
	outlet  channel.Outleter
	config  *config
	log     *logp.Logger
}

// NewInput creates a new TCP input
func NewInput(
	cfg *common.Config,
	outlet channel.Factory,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Experimental("TCP input type is used")

	out, err := outlet(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)

	config := defaultConfig
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	cb := func(data []byte, metadata tcp.Metadata) {
		event := createEvent(data, metadata)
		forwarder.Send(event)
	}

	server, err := tcp.New(cb, &config.Config)
	if err != nil {
		return nil, err
	}

	return &Input{
		server:  server,
		started: atomic.MakeBool(false),
		outlet:  out,
		config:  &config,
		log:     logp.NewLogger("tcp input").With(config.Config.Host),
	}, nil
}

// Run start a TCP input
func (p *Input) Run() {
	if !p.started.Load() {
		p.log.Info("Starting TCP input")
		err := p.server.Start()
		if err != nil {
			p.log.Errorw("Error starting the TCP server", "error", err)
		}
		p.started.Swap(true)
	}
}

// Stop stops TCP server
func (p *Input) Stop() {
	p.log.Info("Stopping TCP input")
	defer p.outlet.Close()
	defer p.started.Swap(false)
	p.server.Stop()
}

// Wait stop the current server
func (p *Input) Wait() {
	p.Stop()
}

func createEvent(raw []byte, metadata tcp.Metadata) *util.Data {
	data := util.NewData()
	data.Event = beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": string(raw),
			"source":  metadata.RemoteAddr.String(),
		},
	}
	return data
}
