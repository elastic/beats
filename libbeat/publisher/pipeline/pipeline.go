// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package pipeline combines all publisher functionality (processors, queue,
// outputs) to create instances of complete publisher pipelines, beats can
// connect to publish events to.
package pipeline

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/diskqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Pipeline implementation providint all beats publisher functionality.
// The pipeline consists of clients, processors, a central queue, an output
// controller and the actual outputs.
// The queue implementing the queue.Queue interface is the most central entity
// to the pipeline, providing support for pushung, batching and pulling events.
// The pipeline adds different ACKing strategies and wait close support on top
// of the queue. For handling ACKs, the pipeline keeps track of filtered out events,
// to be ACKed to the client in correct order.
// The output controller configures a (potentially reloadable) set of load
// balanced output clients. Events will be pulled from the queue and pushed to
// the output clients using a shared work queue for the active outputs.Group.
// Processors in the pipeline are executed in the clients go-routine, before
// entering the queue. No filtering/processing will occur on the output side.
//
// For client connecting to this pipeline, the default PublishMode is
// OutputChooses.
type Pipeline struct {
	beatInfo beat.Info

	monitors Monitors

	outputController *outputController

	observer observer

	// If waitCloseTimeout is positive, then the pipeline will wait up to the
	// specified time when it is closed for pending events to be acknowledged.
	waitCloseTimeout time.Duration

	processors processing.Supporter

	clientTracker *clientTracker

	processorReloader *GlobalProcessorReloader
}

// Settings is used to pass additional settings to a newly created pipeline instance.
type Settings struct {
	// WaitClose sets the maximum duration to block when clients or pipeline itself is closed.
	// When and how WaitClose is applied depends on WaitCloseMode.
	WaitClose time.Duration

	WaitCloseMode WaitCloseMode

	Processors processing.Supporter

	InputQueueSize int
}

type clientTracker struct {
	log *logp.Logger
	// clients holds the pointers to all the connected clients
	clients   map[*client]beat.ClientConfig
	clientsMx sync.Mutex
}

func (ct *clientTracker) RegisterClient(clt *client, cfg beat.ClientConfig) {
	ct.log.Debug("Registering new client %x", clt)
	ct.clientsMx.Lock()
	defer ct.clientsMx.Unlock()
	ct.clients[clt] = cfg
	ct.log.Debug("Registered new client %x", clt)
}

func (ct *clientTracker) UnregisterClient(clt *client) {
	ct.log.Debug("Unregistering client %x", clt)
	ct.clientsMx.Lock()
	defer ct.clientsMx.Unlock()
	delete(ct.clients, clt)
	ct.log.Debug("Unregistered client %x", clt)
}

type clientAction func(clt *client, cfg beat.ClientConfig) error

func (ct *clientTracker) ApplyToAllClients(action clientAction) error {
	ct.clientsMx.Lock()
	defer ct.clientsMx.Unlock()

	for clt, cfg := range ct.clients {
		if clt == nil {
			// should never happen
			ct.log.Warn("encountered nil client pointer while iterating over connected pipeline clients. Skipping...")
			continue
		}
		err := action(clt, cfg)
		if err != nil {
			return fmt.Errorf("error applying action to clients: %w", err)
		}
	}
	return nil
}

type GlobalProcessorReloader struct {
	p *Pipeline
}

func (g *GlobalProcessorReloader) Reload(config *reload.ConfigWithMeta) error {
	fmt.Fprintf(os.Stderr, "GlobalProcessorReloader %v should reload global processors with %v", g, config.Config)
	return nil
}

// WaitCloseMode enumerates the possible behaviors of WaitClose in a pipeline.
type WaitCloseMode uint8

const (
	// NoWaitOnClose disable wait close in the pipeline. Clients can still
	// selectively enable WaitClose when connecting to the pipeline.
	NoWaitOnClose WaitCloseMode = iota

	// WaitOnPipelineClose applies WaitClose to the pipeline itself, waiting for outputs
	// to ACK any outstanding events. This is independent of Clients asking for
	// ACK and/or WaitClose. Clients can still optionally configure WaitClose themselves.
	WaitOnPipelineClose
)

// OutputReloader interface, that can be queried from an active publisher pipeline.
// The output reloader can be used to change the active output.
type OutputReloader interface {
	Reload(
		cfg *reload.ConfigWithMeta,
		factory func(outputs.Observer, conf.Namespace) (outputs.Group, error),
	) error
}

// New create a new Pipeline instance from a queue instance and a set of outputs.
// The new pipeline will take ownership of queue and outputs. On Close, the
// queue and outputs will be closed.
func New(
	beatInfo beat.Info,
	monitors Monitors,
	userQueueConfig conf.Namespace,
	out outputs.Group,
	settings Settings,
) (*Pipeline, error) {
	if monitors.Logger == nil {
		monitors.Logger = logp.NewLogger("publish")
	}

	p := &Pipeline{
		beatInfo:         beatInfo,
		monitors:         monitors,
		observer:         nilObserver,
		waitCloseTimeout: settings.WaitClose,
		processors:       settings.Processors,
		clientTracker: &clientTracker{
			log:       monitors.Logger,
			clients:   make(map[*client]beat.ClientConfig),
			clientsMx: sync.Mutex{},
		},
	}
	if settings.WaitCloseMode == WaitOnPipelineClose && settings.WaitClose > 0 {
		p.waitCloseTimeout = settings.WaitClose
	}

	if monitors.Metrics != nil {
		p.observer = newMetricsObserver(monitors.Metrics)
	}

	// Convert the raw queue config to a parsed Settings object that will
	// be used during queue creation. This lets us fail immediately on startup
	// if there's a configuration problem.
	queueType := defaultQueueType
	if b := userQueueConfig.Name(); b != "" {
		queueType = b
	}
	queueFactory, err := queueFactoryForUserConfig(queueType, userQueueConfig.Config())
	if err != nil {
		return nil, err
	}

	output, err := newOutputController(beatInfo, monitors, p.observer, queueFactory, settings.InputQueueSize)
	if err != nil {
		return nil, err
	}
	p.outputController = output
	p.outputController.Set(out)

	return p, nil
}

// Close stops the pipeline, outputs and queue.
// If WaitClose with WaitOnPipelineClose mode is configured, Close will block
// for a duration of WaitClose, if there are still active events in the pipeline.
// Note: clients must be closed before calling Close.
func (p *Pipeline) Close() error {
	log := p.monitors.Logger

	log.Debug("close pipeline")

	// Note: active clients are not closed / disconnected.
	p.outputController.WaitClose(p.waitCloseTimeout)

	p.observer.cleanup()
	return nil
}

// Connect creates a new client with default settings.
func (p *Pipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

// ConnectWith create a new Client for publishing events to the pipeline.
// The client behavior on close and ACK handling can be configured by setting
// the appropriate fields in the passed ClientConfig.
// If not set otherwise the defaut publish mode is OutputChooses.
//
// It is responsibility of the caller to close the client.
func (p *Pipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	var (
		canDrop    bool
		eventFlags publisher.EventFlags
	)

	err := validateClientConfig(&cfg)
	if err != nil {
		return nil, err
	}

	switch cfg.PublishMode {
	case beat.GuaranteedSend:
		eventFlags = publisher.GuaranteedSend
	case beat.DropIfFull:
		canDrop = true
	}

	waitClose := cfg.WaitClose

	processors, err := p.createEventProcessing(cfg.Processing, publishDisabled)
	if err != nil {
		return nil, err
	}

	clt := &client{
		logger:         p.monitors.Logger,
		isOpen:         atomic.MakeBool(true),
		clientListener: cfg.ClientListener,
		processors:     processors,
		eventFlags:     eventFlags,
		canDrop:        canDrop,
		observer:       p.observer,
		unregisterer:   p.clientTracker,
	}

	ackHandler := cfg.EventListener

	var waiter *clientCloseWaiter
	if waitClose > 0 {
		waiter = newClientCloseWaiter(waitClose)
		if ackHandler == nil {
			ackHandler = waiter
		} else {
			ackHandler = acker.Combine(waiter, ackHandler)
		}
	}

	producerCfg := queue.ProducerConfig{
		ACK: func(count int) {
			clt.observer.eventsACKed(count)
			if ackHandler != nil {
				ackHandler.ACKEvents(count)
			}
		},
	}

	if ackHandler == nil {
		ackHandler = acker.Nil()
	}

	clt.eventListener = ackHandler
	clt.waiter = waiter
	clt.producer = p.outputController.queueProducer(producerCfg)
	if clt.producer == nil {
		// This can only happen if the pipeline was shut down while clients
		// were still waiting to connect.
		return nil, fmt.Errorf("client failed to connect because the pipeline is shutting down")
	}

	p.clientTracker.RegisterClient(clt, cfg)

	p.observer.clientConnected()

	return clt, nil
}

func (p *Pipeline) createEventProcessing(cfg beat.ProcessingConfig, noPublish bool) (beat.Processor, error) {
	if p.processors == nil {
		return nil, nil
	}
	return p.processors.Create(cfg, noPublish)
}

// OutputReloader returns a reloadable object for the output section of this pipeline
func (p *Pipeline) OutputReloader() OutputReloader {
	return p.outputController
}

func (p *Pipeline) GlobalProcessorsReloader() reload.Reloadable {
	return p.processorReloader
}

// Parses the given config and returns a QueueFactory based on it.
// This helper exists to frontload config parsing errors: if there is an
// error in the queue config, we want it to show up as fatal during
// initialization, even if the queue itself isn't created until later.
func queueFactoryForUserConfig(queueType string, userConfig *conf.C) (queue.QueueFactory, error) {
	switch queueType {
	case memqueue.QueueType:
		settings, err := memqueue.SettingsForUserConfig(userConfig)
		if err != nil {
			return nil, err
		}
		return memqueue.FactoryForSettings(settings), nil
	case diskqueue.QueueType:
		settings, err := diskqueue.SettingsForUserConfig(userConfig)
		if err != nil {
			return nil, err
		}
		return diskqueue.FactoryForSettings(settings), nil
	default:
		return nil, fmt.Errorf("unrecognized queue type '%v'", queueType)
	}
}
