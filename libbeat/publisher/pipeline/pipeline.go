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
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/processing"
	"github.com/elastic/beats/libbeat/publisher/queue"
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

	// pipeline ack
	ackMode    pipelineACKMode
	ackActive  atomic.Bool
	ackDone    chan struct{}
	ackBuilder ackBuilder
	eventSema  *sema

	processors processing.Supporter
}

// Settings is used to pass additional settings to a newly created pipeline instance.
type Settings struct {
	// WaitClose sets the maximum duration to block when clients or pipeline itself is closed.
	// When and how WaitClose is applied depends on WaitCloseMode.
	WaitClose time.Duration

	WaitCloseMode WaitCloseMode

	Processors processing.Supporter
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
	cb        *pipelineEventCB
}

type waitCloser struct {
	// keep track of total number of active events (minus dropped by processors)
	events sync.WaitGroup
}

type queueFactory func(queue.Eventer) (queue.Queue, error)

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
	p.ackBuilder = &pipelineEmptyACK{p}
	p.ackActive = atomic.MakeBool(true)

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

	if count := p.queue.BufferConfig().Events; count > 0 {
		p.eventSema = newSema(count)
	}

	maxEvents := p.queue.BufferConfig().Events
	if maxEvents <= 0 {
		// Maximum number of events until acker starts blocking.
		// Only active if pipeline can drop events.
		maxEvents = 64000
	}
	p.eventSema = newSema(maxEvents)

	p.output = newOutputController(beat, monitors, p.observer, p.queue)
	p.output.Set(out)

	return p, nil
}

// SetACKHandler sets a global ACK handler on all events published to the pipeline.
// SetACKHandler must be called before any connection is made.
func (p *Pipeline) SetACKHandler(handler beat.PipelineACKHandler) error {
	p.eventer.mutex.Lock()
	defer p.eventer.mutex.Unlock()

	if !p.eventer.modifyable {
		return errors.New("can not set ack handler on already active pipeline")
	}

	// TODO: check only one type being configured

	cb, err := newPipelineEventCB(handler)
	if err != nil {
		return err
	}

	if cb == nil {
		p.ackBuilder = &pipelineEmptyACK{p}
		p.eventer.cb = nil
		return nil
	}

	p.eventer.cb = cb
	if cb.mode == countACKMode {
		p.ackBuilder = &pipelineCountACK{
			pipeline: p,
			cb:       cb.onCounts,
		}
	} else {
		p.ackBuilder = &pipelineEventsACK{
			pipeline: p,
			cb:       cb.onEvents,
		}
	}

	return nil
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
	return nil
}

// Connect creates a new client with default settings
func (p *Pipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

// ConnectWith create a new Client for publishing events to the pipeline.
// The client behavior on close and ACK handling can be configured by setting
// the appropriate fields in the passed ClientConfig.
func (p *Pipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	var (
		canDrop      bool
		dropOnCancel bool
		eventFlags   publisher.EventFlags
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
		dropOnCancel = true
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
	acker := p.makeACKer(processors != nil, &cfg, waitClose)
	producerCfg := queue.ProducerConfig{
		// Cancel events from queue if acker is configured
		// and no pipeline-wide ACK handler is registered.
		DropOnCancel: dropOnCancel && acker != nil && p.eventer.cb == nil,
	}

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

	if acker != nil {
		producerCfg.ACK = acker.ackEvents
	} else {
		acker = nilACKer
	}

	producer := p.queue.Producer(producerCfg)
	client := &client{
		pipeline:     p,
		isOpen:       atomic.MakeBool(true),
		eventer:      cfg.Events,
		processors:   processors,
		producer:     producer,
		acker:        acker,
		eventFlags:   eventFlags,
		canDrop:      canDrop,
		reportEvents: reportEvents,
	}

	p.observer.clientConnected()
	return client, nil
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
	if e.cb != nil {
		e.cb.reportQueueACK(n)
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
