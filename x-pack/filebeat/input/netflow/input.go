// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"bytes"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	inputName = "netflow"
)

var (
	numPackets  = monitoring.NewUint(nil, "filebeat.input.netflow.packets.received")
	numDropped  = monitoring.NewUint(nil, "filebeat.input.netflow.packets.dropped")
	numFlows    = monitoring.NewUint(nil, "filebeat.input.netflow.flows")
	aliveInputs atomic.Int
	logger      *logp.Logger
	initLogger  sync.Once
)

type packet struct {
	data   []byte
	source net.Addr
}

type netflowInput struct {
	mutex            sync.Mutex
	udp              *udp.Server
	decoder          *decoder.Decoder
	outlet           channel.Outleter
	forwarder        *harvester.Forwarder
	internalNetworks []string
	logger           *logp.Logger
	queueC           chan packet
	queueSize        int
	started          bool
}

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(err)
	}
}

// An adapter so that logp.Logger can be used as a log.Logger.
type logDebugWrapper struct {
	sync.Mutex
	Logger *logp.Logger
	buf    []byte
}

// Write writes messages to the log.
func (w *logDebugWrapper) Write(p []byte) (n int, err error) {
	w.Lock()
	defer w.Unlock()
	n = len(p)
	w.buf = append(w.buf, p...)
	for endl := bytes.IndexByte(w.buf, '\n'); endl != -1; endl = bytes.IndexByte(w.buf, '\n') {
		w.Logger.Debug(string(w.buf[:endl]))
		w.buf = w.buf[endl+1:]
	}
	return n, nil
}

// NewInput creates a new Netflow input
func NewInput(
	cfg *conf.C,
	connector channel.Connector,
	context input.Context,
) (input.Input, error) {
	initLogger.Do(func() {
		logger = logp.NewLogger(inputName)
	})
	out, err := connector.Connect(cfg)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		out.Close()
		return nil, err
	}

	var customFields []fields.FieldDict
	for _, yamlPath := range config.CustomDefinitions {
		f, err := LoadFieldDefinitionsFromFile(yamlPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parsing custom field definitions from file '%s'", yamlPath)
		}
		customFields = append(customFields, f)
	}
	decoder, err := decoder.NewDecoder(decoder.NewConfig().
		WithProtocols(config.Protocols...).
		WithExpiration(config.ExpirationTimeout).
		WithLogOutput(&logDebugWrapper{Logger: logger}).
		WithCustomFields(customFields...).
		WithSequenceResetEnabled(config.DetectSequenceReset))
	if err != nil {
		return nil, errors.Wrapf(err, "error initializing netflow decoder")
	}

	input := &netflowInput{
		outlet:           out,
		internalNetworks: config.InternalNetworks,
		forwarder:        harvester.NewForwarder(out),
		decoder:          decoder,
		logger:           logger,
		queueSize:        config.PacketQueueSize,
	}

	input.udp = udp.New(&config.Config, input.packetDispatch)
	return input, nil
}

func (p *netflowInput) Publish(events []beat.Event) error {
	for _, evt := range events {
		p.forwarder.Send(evt)
	}
	return nil
}

// Run starts listening for NetFlow events over the network.
func (p *netflowInput) Run() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.started {
		logger.Info("Starting UDP input")

		if err := p.decoder.Start(); err != nil {
			logger.Errorw("Failed to start netflow decoder", "error", err)
			p.outlet.Close()
			return
		}

		p.queueC = make(chan packet, p.queueSize)
		err := p.udp.Start()
		if err != nil {
			logger.Errorf("Error running harvester: %v", err)
			p.outlet.Close()
			p.decoder.Stop()
			close(p.queueC)
			return
		}

		go p.recvRoutine()
		// Only the first active input launches the stats thread
		if aliveInputs.Inc() == 1 && logger.IsDebug() {
			go p.statsLoop()
		}
		p.started = true
	}
}

// Stop stops the UDP input
func (p *netflowInput) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.started {
		aliveInputs.Dec()
		defer p.outlet.Close()
		defer close(p.queueC)

		logger.Info("Stopping UDP input")
		p.udp.Stop()
		p.started = false
	}
}

// Wait suspends the UDP input
func (p *netflowInput) Wait() {
	p.Stop()
}

func (p *netflowInput) statsLoop() {
	prevPackets := numPackets.Get()
	prevFlows := numFlows.Get()
	prevDropped := numDropped.Get()
	// The stats thread only monitors queue length for the first input
	prevQueue := len(p.queueC)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		packets := numPackets.Get()
		flows := numFlows.Get()
		dropped := numDropped.Get()
		queue := len(p.queueC)
		if packets > prevPackets || flows > prevFlows || dropped > prevDropped || queue > prevQueue {
			logger.Debugf("Stats total:[ packets=%d dropped=%d flows=%d queue_len=%d ] delta:[ packets/s=%d dropped/s=%d flows/s=%d queue_len/s=%+d ]",
				packets, dropped, flows, queue, packets-prevPackets, dropped-prevDropped, flows-prevFlows, queue-prevQueue)
			prevFlows = flows
			prevPackets = packets
			prevQueue = queue
			prevDropped = dropped
		} else {
			p.mutex.Lock()
			count := aliveInputs.Load()
			p.mutex.Unlock()
			if count == 0 {
				break
			}
		}
	}
}

func (p *netflowInput) packetDispatch(data []byte, metadata inputsource.NetworkMetadata) {
	select {
	case p.queueC <- packet{data, metadata.RemoteAddr}:
		numPackets.Inc()
	default:
		numDropped.Inc()
	}
}

func (p *netflowInput) recvRoutine() {
	for packet := range p.queueC {
		flows, err := p.decoder.Read(bytes.NewBuffer(packet.data), packet.source)
		if err != nil {
			p.logger.Warnf("Error parsing NetFlow packet of length %d from %s: %v", len(packet.data), packet.source, err)
		}
		if n := len(flows); n > 0 {
			evs := make([]beat.Event, n)
			numFlows.Add(uint64(n))
			for i, flow := range flows {
				evs[i] = toBeatEvent(flow, p.internalNetworks)
			}
			p.Publish(evs)
		}
	}
}
