// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName = "netflow"
)

var (
	numPackets  = monitoring.NewUint(nil, "filebeat.input.netflow.packets.received")
	numDropped  = monitoring.NewUint(nil, "filebeat.input.netflow.packets.dropped")
	numFlows    = monitoring.NewUint(nil, "filebeat.input.netflow.flows")
	aliveInputs atomic.Int
)

func Plugin(log *logp.Logger) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "collect and decode packets of netflow protocol",
		Manager: &netflowInputManager{
			log: log.Named(inputName),
		},
	}
}

type netflowInputManager struct {
	log *logp.Logger
}

func (im *netflowInputManager) Init(_ unison.Group, _ v2.Mode) error {
	return nil
}

func (im *netflowInputManager) Create(cfg *conf.C) (v2.Input, error) {
	inputCfg := defaultConfig
	if err := cfg.Unpack(&inputCfg); err != nil {
		return nil, err
	}

	customFields := make([]fields.FieldDict, len(inputCfg.CustomDefinitions))
	for idx, yamlPath := range inputCfg.CustomDefinitions {
		f, err := LoadFieldDefinitionsFromFile(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed parsing custom field definitions from file '%s': %w", yamlPath, err)
		}
		customFields[idx] = f
	}

	dec, err := decoder.NewDecoder(decoder.NewConfig().
		WithProtocols(inputCfg.Protocols...).
		WithExpiration(inputCfg.ExpirationTimeout).
		WithLogOutput(&logDebugWrapper{Logger: im.log}).
		WithCustomFields(customFields...).
		WithSequenceResetEnabled(inputCfg.DetectSequenceReset).
		WithSharedTemplates(inputCfg.ShareTemplates))
	if err != nil {
		return nil, fmt.Errorf("error initializing netflow decoder: %w", err)
	}

	input := &netflowInput{
		decoder:          dec,
		internalNetworks: inputCfg.InternalNetworks,
		logger:           im.log,
		queueSize:        inputCfg.PacketQueueSize,
	}

	input.udp = udp.New(&inputCfg.Config, input.packetDispatch)

	return input, nil
}

type packet struct {
	data   []byte
	source net.Addr
}

type netflowInput struct {
	mtx              sync.Mutex
	udp              *udp.Server
	decoder          *decoder.Decoder
	client           beat.Client
	internalNetworks []string
	logger           *logp.Logger
	queueC           chan packet
	queueSize        int
	started          bool
}

func (n *netflowInput) Name() string {
	return inputName
}

func (n *netflowInput) Test(_ v2.TestContext) error {
	return nil
}

func (n *netflowInput) packetDispatch(data []byte, metadata inputsource.NetworkMetadata) {
	select {
	case n.queueC <- packet{data, metadata.RemoteAddr}:
		numPackets.Inc()
	default:
		numDropped.Inc()
	}
}

func (n *netflowInput) Run(context v2.Context, connector beat.PipelineConnector) error {
	n.mtx.Lock()
	if n.started {
		n.mtx.Unlock()
		return nil
	}

	n.started = true
	n.mtx.Unlock()

	n.logger.Info("Starting netflow input")

	n.logger.Info("Connecting to beat event publishing")
	client, err := connector.ConnectWith(beat.ClientConfig{
		PublishMode: beat.DefaultGuarantees,
		Processing: beat.ProcessingConfig{
			EventNormalization: boolPtr(true),
		},
		CloseRef:      context.Cancelation,
		EventListener: nil,
	})
	if err != nil {
		n.logger.Errorw("Failed connecting to beat event publishing", "error", err)
		n.stop()
		return err
	}

	n.logger.Info("Starting netflow decoder")
	if err := n.decoder.Start(); err != nil {
		n.logger.Errorw("Failed to start netflow decoder", "error", err)
		n.stop()
		return err
	}

	n.queueC = make(chan packet, n.queueSize)

	n.logger.Info("Starting udp server")
	err = n.udp.Start()
	if err != nil {
		n.logger.Errorf("Failed to start udp server: %v", err)
		n.stop()
		return err
	}

	if aliveInputs.Inc() == 1 && n.logger.IsDebug() {
		go n.statsLoop(ctxtool.FromCanceller(context.Cancelation))
	}
	defer aliveInputs.Dec()

	go func() {
		<-context.Cancelation.Done()
		n.stop()
	}()

	for packet := range n.queueC {
		flows, err := n.decoder.Read(bytes.NewBuffer(packet.data), packet.source)
		if err != nil {
			n.logger.Warnf("Error parsing NetFlow packet of length %d from %s: %v", len(packet.data), packet.source, err)
			continue
		}

		fLen := len(flows)
		if fLen == 0 {
			continue
		}
		evs := make([]beat.Event, fLen)
		numFlows.Add(uint64(fLen))
		for i, flow := range flows {
			evs[i] = toBeatEvent(flow, n.internalNetworks)
		}
		client.PublishAll(evs)
	}

	return nil
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

// stop stops the netflow input
func (n *netflowInput) stop() {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	if !n.started {
		return
	}

	if n.udp != nil {
		n.udp.Stop()
	}

	if n.decoder != nil {
		if err := n.decoder.Stop(); err != nil {
			n.logger.Errorw("Error stopping decoder", "error", err)
		}
	}

	if n.client != nil {
		if err := n.client.Close(); err != nil {
			n.logger.Errorw("Error closing beat client", "error", err)
		}
	}

	close(n.queueC)

	n.started = false
}

func (n *netflowInput) statsLoop(ctx context.Context) {
	prevPackets := numPackets.Get()
	prevFlows := numFlows.Get()
	prevDropped := numDropped.Get()
	// The stats thread only monitors queue length for the first input
	prevQueue := len(n.queueC)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			packets := numPackets.Get()
			flows := numFlows.Get()
			dropped := numDropped.Get()
			queue := len(n.queueC)
			if packets > prevPackets || flows > prevFlows || dropped > prevDropped || queue > prevQueue {
				n.logger.Debugf("Stats total:[ packets=%d dropped=%d flows=%d queue_len=%d ] delta:[ packets/s=%d dropped/s=%d flows/s=%d queue_len/s=%+d ]",
					packets, dropped, flows, queue, packets-prevPackets, dropped-prevDropped, flows-prevFlows, queue-prevQueue)
				prevFlows = flows
				prevPackets = packets
				prevQueue = queue
				prevDropped = dropped
				continue
			}

			n.mtx.Lock()
			count := aliveInputs.Load()
			n.mtx.Unlock()
			if count == 0 {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func boolPtr(b bool) *bool { return &b }
