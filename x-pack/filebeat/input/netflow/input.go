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

	"github.com/elastic/beats/v7/filebeat/input/netmetrics"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

const (
	inputName = "netflow"
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

func (im *netflowInputManager) Init(_ unison.Group) error {
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

	input := &netflowInput{
		cfg:              inputCfg,
		internalNetworks: inputCfg.InternalNetworks,
		logger:           im.log,
		queueSize:        inputCfg.PacketQueueSize,
	}

	return input, nil
}

type packet struct {
	data   []byte
	source net.Addr
}

type netflowInput struct {
	mtx              sync.Mutex
	cfg              config
	decoder          *decoder.Decoder
	clients          []beat.Client
	metrics          *netflowMetrics
	udpMetrics       *netmetrics.UDP
	customFields     []fields.FieldDict
	internalNetworks []string
	logger           *logp.Logger
	queueC           chan packet
	wg               sync.WaitGroup
	ctx              context.Context
	cancelFunc       context.CancelFunc
	queueSize        int
	started          bool
}

func (n *netflowInput) Name() string {
	return inputName
}

func (n *netflowInput) Test(_ v2.TestContext) error {
	return nil
}

func (n *netflowInput) Run(env v2.Context, connector beat.PipelineConnector) error {
	n.mtx.Lock()
	if n.started {
		n.mtx.Unlock()
		return nil
	}
	n.ctx, n.cancelFunc = context.WithCancel(v2.GoContextFromCanceler(env.Cancelation))
	n.started = true
	n.mtx.Unlock()

	env.UpdateStatus(status.Starting, "Starting netflow input")
	n.logger.Info("Starting netflow input")

	n.logger.Info("Connecting to beat event publishing")

	const pollInterval = time.Minute
	n.udpMetrics = netmetrics.NewUDP("netflow", env.ID, n.cfg.Host, uint64(n.cfg.ReadBuffer), pollInterval, n.logger)
	defer n.udpMetrics.Close()

	n.metrics = newInputMetrics(n.udpMetrics.Registry())
	var err error
	n.decoder, err = decoder.NewDecoder(decoder.NewConfig().
		WithProtocols(n.cfg.Protocols...).
		WithExpiration(n.cfg.ExpirationTimeout).
		WithLogOutput(&logDebugWrapper{Logger: n.logger}).
		WithCustomFields(n.customFields...).
		WithSequenceResetEnabled(n.cfg.DetectSequenceReset).
		WithSharedTemplates(n.cfg.ShareTemplates).
		WithActiveSessionsMetric(n.metrics.ActiveSessions()).
		WithCache(n.cfg.NumberOfWorkers > 1))
	if err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to initialize netflow decoder: %v", err))
		return fmt.Errorf("error initializing netflow decoder: %w", err)
	}

	n.logger.Info("Starting netflow decoder")
	if err := n.decoder.Start(); err != nil {
		env.UpdateStatus(status.Failed, fmt.Sprintf("Failed to start netflow decoder: %v", err))
		n.logger.Errorw("Failed to start netflow decoder", "error", err)
		n.stop()
		return err
	}

	n.queueC = make(chan packet, n.queueSize)
	for i := uint32(0); i < n.cfg.NumberOfWorkers; i++ {
		client, err := connector.ConnectWith(beat.ClientConfig{
			PublishMode: beat.DefaultGuarantees,
			Processing: beat.ProcessingConfig{
				EventNormalization: boolPtr(true),
			},
			EventListener: nil,
		})
		if err != nil {
			env.UpdateStatus(status.Failed, fmt.Sprintf("Failed connecting to beat event publishing: %v", err))
			n.logger.Errorw("Failed connecting to beat event publishing", "error", err)
			n.stop()
			return err
		}

		n.clients = append(n.clients, client)
		n.wg.Add(1)
		go func(client beat.Client) {
			defer n.wg.Done()
			for {
				select {
				case <-n.ctx.Done():
					return
				case pkt := <-n.queueC:
					pktStartTime := time.Now()
					flows, err := n.decoder.Read(bytes.NewBuffer(pkt.data), pkt.source)
					if err != nil {
						n.logger.Warnf("Error parsing NetFlow packet of length %d from %s: %v", len(pkt.data), pkt.source, err)
						if decodeErrors := n.metrics.DecodeErrors(); decodeErrors != nil {
							decodeErrors.Inc()
						}
						continue
					}

					fLen := len(flows)
					if fLen != 0 {
						evs := make([]beat.Event, fLen)
						if flowsTotal := n.metrics.Flows(); flowsTotal != nil {
							flowsTotal.Add(uint64(fLen))
						}
						for flowIdx, flow := range flows {
							evs[flowIdx] = toBeatEvent(flow, n.internalNetworks)
						}
						client.PublishAll(evs)
					}
					n.udpMetrics.Log(pkt.data, pktStartTime)
				}
			}
		}(client)
	}

	n.logger.Info("Starting udp server")

	udpServer := udp.New(&n.cfg.Config, func(data []byte, metadata inputsource.NetworkMetadata) {
		if n.ctx.Err() != nil {
			return
		}
		select {
		case <-n.ctx.Done():
			return
		case n.queueC <- packet{data, metadata.RemoteAddr}:
		default:
			if discardedEvents := n.metrics.DiscardedEvents(); discardedEvents != nil {
				discardedEvents.Inc()
			}
		}
	})
	err = udpServer.Start()
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to start udp server: %v", err)
		n.logger.Errorf(errorMsg)
		env.UpdateStatus(status.Failed, errorMsg)
		n.stop()
		return err
	}
	defer udpServer.Stop()

	env.UpdateStatus(status.Running, "")
	<-n.ctx.Done()
	n.stop()

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

	n.cancelFunc()
	n.wg.Wait()

	if !n.started {
		return
	}

	if n.decoder != nil {
		if err := n.decoder.Stop(); err != nil {
			n.logger.Errorw("Error stopping decoder", "error", err)
		}
		n.decoder = nil
	}

	if n.clients != nil {
		for _, client := range n.clients {
			if err := client.Close(); err != nil {
				n.logger.Errorw("Error closing beat client", "error", err)
			}
		}
	}

	close(n.queueC)

	n.started = false
}

func boolPtr(b bool) *bool { return &b }
