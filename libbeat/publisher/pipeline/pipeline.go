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
	"reflect"
	"sync"
	"time"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/acker"
	"github.com/elastic/beats/v8/libbeat/common/atomic"
	"github.com/elastic/beats/v8/libbeat/common/reload"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/outputs"
	"github.com/elastic/beats/v8/libbeat/publisher"
	"github.com/elastic/beats/v8/libbeat/publisher/processing"
	"github.com/elastic/beats/v8/libbeat/publisher/queue"
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

	queue  queue.Queue
	output *outputController

	observer observer

	eventer pipelineEventer

	// wait close support
	waitCloseMode    WaitCloseMode
	waitCloseTimeout time.Duration
	waitCloser       *waitCloser

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

	// WaitOnClientClose applies WaitClose timeout to each client connecting to
	// the pipeline. Clients are still allowed to overwrite WaitClose with a timeout > 0s.
	WaitOnClientClose
)

// OutputReloader interface, that can be queried from an active publisher pipeline.
// The output reloader can be used to change the active output.
type OutputReloader interface {
	Reload(
		cfg *reload.ConfigWithMeta,
		factory func(outputs.Observer, common.ConfigNamespace) (outputs.Group, error),
	) error
}

type pipelineEventer struct {
	mutex      sync.Mutex
	modifyable bool

	observer  queueObserver
	waitClose *waitCloser
}

type waitCloser struct {
	// keep track of total number of active events (minus dropped by processors)
	events sync.WaitGroup
}

type queueFactory func(queue.ACKListener) (queue.Queue, error)

// New create a new Pipeline instance from a queue instance and a set of outputs.
// The new pipeline will take ownership of queue and outputs. On Close, the
// queue and outputs will be closed.
func New(
	beat beat.Info,
	monitors Monitors,
	queueFactory queueFactory,
	out outputs.Group,
	settings Settings,
) (*Pipeline, error) {
	var err error

	if monitors.Logger == nil {
		monitors.Logger = logp.NewLogger("publish")
	}

	p := &Pipeline{
		beatInfo:         beat,
		monitors:         monitors,
		observer:         nilObserver,
		waitCloseMode:    settings.WaitCloseMode,
		waitCloseTimeout: settings.WaitClose,
		processors:       settings.Processors,
	}

	if monitors.Metrics != nil {
		p.observer = newMetricsObserver(monitors.Metrics)
	}
	p.eventer.observer = p.observer
	p.eventer.modifyable = true

	if settings.WaitCloseMode == WaitOnPipelineClose && settings.WaitClose > 0 {
		p.waitCloser = &waitCloser{}

		// waitCloser decrements counter on queue ACK (not per client)
		p.eventer.waitClose = p.waitCloser
	}

	p.queue, err = queueFactory(&p.eventer)
	if err != nil {
		return nil, err
	}

	maxEvents := p.queue.BufferConfig().MaxEvents
	if maxEvents <= 0 {
		// Maximum number of events until acker starts blocking.
		// Only active if pipeline can drop events.
		maxEvents = 64000
	}
	p.observer.queueMaxEvents(maxEvents)

	p.output = newOutputController(beat, monitors, p.observer, p.queue)
	p.output.Set(out)

	return p, nil
}

// Close stops the pipeline, outputs and queue.
// If WaitClose with WaitOnPipelineClose mode is configured, Close will block
// for a duration of WaitClose, if there are still active events in the pipeline.
// Note: clients must be closed before calling Close.
func (p *Pipeline) Close() error {
	log := p.monitors.Logger

	log.Debug("close pipeline")

	if p.waitCloser != nil {
		ch := make(chan struct{})
		go func() {
			p.waitCloser.wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			// all events have been ACKed

		case <-time.After(p.waitCloseTimeout):
			// timeout -> close pipeline with pending events
		}

	}

	// TODO: close/disconnect still active clients

	// close output before shutting down queue
	p.output.Close()

	// shutdown queue
	err := p.queue.Close()
	if err != nil {
		log.Error("pipeline queue shutdown error: ", err)
	}

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

	p.eventer.mutex.Lock()
	p.eventer.modifyable = false
	p.eventer.mutex.Unlock()

	switch cfg.PublishMode {
	case beat.GuaranteedSend:
		eventFlags = publisher.GuaranteedSend
	case beat.DropIfFull:
		canDrop = true
	}

	waitClose := cfg.WaitClose
	reportEvents := p.waitCloser != nil

	switch p.waitCloseMode {
	case NoWaitOnClose:

	case WaitOnClientClose:
		if waitClose <= 0 {
			waitClose = p.waitCloseTimeout
		}
	}

	processors, err := p.createEventProcessing(cfg.Processing, publishDisabled)
	if err != nil {
		return nil, err
	}

	client := &client{
		pipeline:     p,
		closeRef:     cfg.CloseRef,
		done:         make(chan struct{}),
		isOpen:       atomic.MakeBool(true),
		eventer:      cfg.Events,
		processors:   processors,
		eventFlags:   eventFlags,
		canDrop:      canDrop,
		reportEvents: reportEvents,
	}

	ackHandler := cfg.ACKHandler

	producerCfg := queue.ProducerConfig{}

	if reportEvents || cfg.Events != nil {
		producerCfg.OnDrop = func(event beat.Event) {
			if cfg.Events != nil {
				cfg.Events.DroppedOnPublish(event)
			}
			if reportEvents {
				p.waitCloser.dec(1)
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

	client.acker = ackHandler
	client.waiter = waiter
	client.producer = p.queue.Producer(producerCfg)

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
			client := recv.Interface().(*client)
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

func (e *pipelineEventer) OnACK(n int) {
	e.observer.queueACKed(n)

	if wc := e.waitClose; wc != nil {
		wc.dec(n)
	}
}

func (e *waitCloser) inc() {
	e.events.Add(1)
}

func (e *waitCloser) dec(n int) {
	for i := 0; i < n; i++ {
		e.events.Done()
	}
}

func (e *waitCloser) wait() {
	e.events.Wait()
}

// OutputReloader returns a reloadable object for the output section of this pipeline
func (p *Pipeline) OutputReloader() OutputReloader {
	return p.output
}
