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
	"reflect"
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

	// wait close support. If eventWaitGroup is non-nil, then publishing
	// an event through this pipeline will increment it and acknowledging
	// a published event will decrement it, so the pipeline can wait on
	// the group on shutdown to allow pending events to be acknowledged.
	waitCloseTimeout time.Duration
	eventWaitGroup   *sync.WaitGroup

	// closeRef signal propagation support
	guardStartSigPropagation sync.Once
	sigNewClient             chan *client

	processors processing.Supporter
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
	beat beat.Info,
	monitors Monitors,
	userQueueConfig conf.Namespace,
	out outputs.Group,
	settings Settings,
) (*Pipeline, error) {
	if monitors.Logger == nil {
		monitors.Logger = logp.NewLogger("publish")
	}

	p := &Pipeline{
		beatInfo:         beat,
		monitors:         monitors,
		observer:         nilObserver,
		waitCloseTimeout: settings.WaitClose,
		processors:       settings.Processors,
	}
	if settings.WaitCloseMode == WaitOnPipelineClose && settings.WaitClose > 0 {
		// If wait-on-close is enabled, give the pipeline a WaitGroup for
		// events that have been Published but not yet ACKed.
		p.eventWaitGroup = &sync.WaitGroup{}
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

	output, err := newOutputController(beat, monitors, p.observer, p.eventWaitGroup, queueFactory, settings.InputQueueSize)
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

	if p.eventWaitGroup != nil {
		ch := make(chan struct{})
		go func() {
			p.eventWaitGroup.Wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			// all events have been ACKed

		case <-time.After(p.waitCloseTimeout):
			// timeout -> close pipeline with pending events
		}
	}

	// Note: active clients are not closed / disconnected.
	p.outputController.Close()

	p.observer.cleanup()
	if p.sigNewClient != nil {
		close(p.sigNewClient)
	}
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

	client := &client{
		logger:         p.monitors.Logger,
		closeRef:       cfg.CloseRef,
		done:           make(chan struct{}),
		isOpen:         atomic.MakeBool(true),
		clientListener: cfg.ClientListener,
		processors:     processors,
		eventFlags:     eventFlags,
		canDrop:        canDrop,
		eventWaitGroup: p.eventWaitGroup,
		observer:       p.observer,
	}

	ackHandler := cfg.EventListener

	producerCfg := queue.ProducerConfig{}

	if client.eventWaitGroup != nil || cfg.ClientListener != nil {
		producerCfg.OnDrop = func(event queue.Entry) {
			publisherEvent, _ := event.(publisher.Event)
			if cfg.ClientListener != nil {
				cfg.ClientListener.DroppedOnPublish(publisherEvent.Content)
			}
			if client.eventWaitGroup != nil {
				client.eventWaitGroup.Add(-1)
			}
		}
	}

	var waiter *clientCloseWaiter
	if waitClose > 0 {
		waiter = newClientCloseWaiter(waitClose)
	}

	if waiter != nil {
		if ackHandler == nil {
			ackHandler = waiter
		} else {
			ackHandler = acker.Combine(waiter, ackHandler)
		}
	}

	if ackHandler != nil {
		producerCfg.ACK = ackHandler.ACKEvents
	} else {
		ackHandler = acker.Nil()
	}

	client.eventListener = ackHandler
	client.waiter = waiter
	client.producer = p.outputController.queueProducer(producerCfg)
	client.encoder = p.outputController.encoder()
	if client.producer == nil {
		// This can only happen if the pipeline was shut down while clients
		// were still waiting to connect.
		return nil, fmt.Errorf("client failed to connect because the pipeline is shutting down")
	}

	p.observer.clientConnected()

	if client.closeRef != nil {
		p.registerSignalPropagation(client)
	}

	return client, nil
}

func (p *Pipeline) registerSignalPropagation(c *client) {
	p.guardStartSigPropagation.Do(func() {
		p.sigNewClient = make(chan *client, 1)
		go p.runSignalPropagation()
	})
	p.sigNewClient <- c
}

func (p *Pipeline) runSignalPropagation() {
	var channels []reflect.SelectCase
	var clients []*client

	channels = append(channels, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(p.sigNewClient),
	})

	for {
		chosen, recv, recvOK := reflect.Select(channels)
		if chosen == 0 {
			if !recvOK {
				// sigNewClient was closed
				return
			}

			// new client -> register client for signal propagation.
			if client := recv.Interface().(*client); client != nil {
				channels = append(channels,
					reflect.SelectCase{
						Dir:  reflect.SelectRecv,
						Chan: reflect.ValueOf(client.closeRef.Done()),
					},
					reflect.SelectCase{
						Dir:  reflect.SelectRecv,
						Chan: reflect.ValueOf(client.done),
					},
				)
				clients = append(clients, client)
			}
			continue
		}

		// find client we received a signal for. If client.done was closed, then
		// we have to remove the client only. But if closeRef did trigger the signal, then
		// we have to propagate the async close to the client.
		// In either case, the client will be removed

		i := (chosen - 1) / 2
		isSig := (chosen & 1) == 1
		if isSig {
			client := clients[i]
			client.Close()
		}

		// remove:
		last := len(clients) - 1
		ch1 := i*2 + 1
		ch2 := ch1 + 1
		lastCh1 := last*2 + 1
		lastCh2 := lastCh1 + 1

		clients[i], clients[last] = clients[last], nil
		channels[ch1], channels[lastCh1] = channels[lastCh1], reflect.SelectCase{}
		channels[ch2], channels[lastCh2] = channels[lastCh2], reflect.SelectCase{}

		clients = clients[:last]
		channels = channels[:lastCh1]
		if cap(clients) > 10 && len(clients) <= cap(clients)/2 {
			clientsTmp := make([]*client, len(clients))
			copy(clientsTmp, clients)
			clients = clientsTmp

			channelsTmp := make([]reflect.SelectCase, len(channels))
			copy(channelsTmp, channels)
			channels = channelsTmp
		}
	}
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
