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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/diskqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/paths"
)

// reaperInterval is how often the reaper re-checks pending clients whose events
// have not drained yet. Finalization is cleanup, not latency-sensitive, so a
// coarse interval keeps the reaper cheap; a client whose events are already
// acked when it is handed over is finalized immediately on the notify wakeup.
const reaperInterval = 50 * time.Millisecond

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

	outputController outputController

	observer observer

	// If waitCloseTimeout is positive, then the pipeline will wait up to the
	// specified time when it is closed for pending events to be acknowledged.
	waitCloseTimeout time.Duration

	// forceCloseQueue causes us to force close the queue after the waitCloseTimeout
	// elapses.
	forceCloseQueue bool

	// Close _shouldn't_ be called multiple times, but handle it gracefully if it does.
	closeOnce sync.Once

	processors processing.Supporter

	// clients is the set of connected clients. The Pipeline finalizes each of
	// them (stage two of client shutdown, client.disconnect) when it is
	// disconnected. Clients register on ConnectWith and remove themselves when
	// disconnected. Guarded by clientsMu.
	clientsMu sync.Mutex
	clients   map[*client]struct{}

	// reaper state. A single goroutine (reapClosedClients) finalizes clients
	// that were Closed while the pipeline keeps running, as soon as their
	// events drain (their producer's ACKWaitChan closes), instead of waiting
	// for the whole pipeline to disconnect. This keeps the clients map, ack
	// handlers and active-client metrics from growing under high client churn.
	// reaperPending holds the clients awaiting drain (guarded by reaperMu);
	// reaperNotify wakes the reaper when the set changes; reaperDone stops it.
	reaperMu      sync.Mutex
	reaperPending map[*client]struct{}
	reaperNotify  chan struct{}
	reaperDone    chan struct{}
	reaperWG      sync.WaitGroup
}

// Settings is used to pass additional settings to a newly created pipeline instance.
type Settings struct {
	// WaitClose sets the maximum duration to block when clients or pipeline itself is closed.
	// When and how WaitClose is applied depends on WaitCloseMode.
	WaitClose time.Duration

	// This field has no effect when running as a Beats receiver.
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

	// WaitOnPipelineCloseThenForce is identical to WaitOnPipelineClose, but it also force closes
	// the queue after the timeout, dropping in-flight data and unprocessed acknowledgements.
	// This is useful when we know terminating the process won't free the memory for us, such as
	// when running in an otel receiver.
	WaitOnPipelineCloseThenForce
)

// outputController is the interface between the Pipeline and the output,
// which may be either the legacy Beats output pipeline (under the process
// runtime) or a bridge to the OTel Collector (when running as a Beats
// receiver under the otel runtime).
type outputController interface {
	// queueProducer creates a queue producer with the given config, blocking
	// until the queue is created if it does not yet exist.
	queueProducer(config queue.ProducerConfig) queue.Producer[publisher.Event]

	// Close the queue and output, waiting for pending events until all are
	// acknowledged or the provided context expires.
	// The force parameter has no effect when running as a Beats receiver.
	waitClose(ctx context.Context, force bool) error
}

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
		monitors.Logger = beat.Logger.Named("publish")
	}

	p := &Pipeline{
		beatInfo:         beat,
		monitors:         monitors,
		observer:         nilObserver,
		waitCloseTimeout: settings.WaitClose,
		processors:       settings.Processors,
		clients:          make(map[*client]struct{}),
	}

	p.forceCloseQueue = settings.WaitCloseMode == WaitOnPipelineCloseThenForce

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
	queueFactory, _, err := queueFactoryForUserConfig(queueType, userQueueConfig.Config(), beat.Paths)
	if err != nil {
		return nil, err
	}

	outputController, err := newProcessOutputController(beat, monitors, p.observer, queueFactory, settings.InputQueueSize)
	if err != nil {
		return nil, err
	}
	outputController.Set(out)
	p.outputController = outputController

	p.startReaper()
	return p, nil
}

func NewForReceiver(
	beatInfo beat.Info,
	monitors Monitors,
	userQueueConfig conf.Namespace,
	settings Settings,
) (*Pipeline, error) {
	p := &Pipeline{
		beatInfo:         beatInfo,
		monitors:         monitors,
		observer:         newMetricsObserver(monitors.Metrics),
		waitCloseTimeout: settings.WaitClose,
		processors:       settings.Processors,
		clients:          make(map[*client]struct{}),
	}

	// Convert the raw queue config to a parsed Settings object that will
	// be used during queue creation. This lets us fail immediately on startup
	// if there's a configuration problem.
	queueType := defaultQueueType
	if b := userQueueConfig.Name(); b != "" {
		queueType = b
	}
	// Receiver pipelines route through the OTel output controller. With an
	// in-memory queue configuration the controller joins the process-global
	// slabqueue pool, sharing one in-memory event budget across all receivers.
	// With an explicit queue.disk config the controller falls back to building
	// its queue via queueFactory and owns it outright.
	queueFactory, queueConfig, err := queueFactoryForUserConfig(queueType, userQueueConfig.Config(), beatInfo.Paths)
	if err != nil {
		return nil, err
	}

	p.outputController, err = newOTelOutputController(beatInfo, monitors, p.observer, queueFactory, queueConfig)
	if err != nil {
		return nil, err
	}

	p.startReaper()
	return p, nil
}

// Disconnect stops the pipeline, outputs and queue.
// If WaitClose with WaitOnPipelineClose mode is configured, Disconnect will block
// for a duration of WaitClose, if there are still active events in the pipeline.
// Note: clients will no longer accept new Publish calls once Disconnect is started,
// and will no longer receive event acknowledgments once Disconnect returns.
//
// The Beater is expected to close its clients (stage one) before disconnecting
// the pipeline; Disconnect then performs stage two for any still-registered
// client — see issues #50104 and #49794.
func (p *Pipeline) Disconnect(ctx context.Context) error {
	p.closeOnce.Do(func() {
		log := p.monitors.Logger

		log.Debug("close pipeline")

		// The Beater determines how long to wait before full disconnection by
		// supplying a context with a deadline (issue #49794). If the caller did
		// not set one, fall back to the pipeline's configured waitCloseTimeout.
		timeoutCtx := ctx
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			timeoutCtx, cancel = context.WithTimeout(context.Background(), p.waitCloseTimeout)
			defer cancel()
		}
		p.outputController.waitClose(timeoutCtx, p.forceCloseQueue)

		// Stage two of client shutdown: the queue has now drained or been
		// force-closed and no further acknowledgments will arrive, so finalize
		// every still-registered client (stop ack handling, drop references).
		p.disconnectClients()

		// Stop the reaper now that all clients are finalized, and wait for it
		// to exit so it does not outlive the pipeline.
		close(p.reaperDone)
		p.reaperWG.Wait()

		p.observer.cleanup()
	})
	return nil
}

// registerClient adds a connected client to the set the Pipeline finalizes on
// Disconnect.
func (p *Pipeline) registerClient(c *client) {
	p.clientsMu.Lock()
	p.clients[c] = struct{}{}
	p.clientsMu.Unlock()
}

// unregisterClient removes a client from the set once it has been disconnected.
// Called from client.disconnect via the onRemove callback.
func (p *Pipeline) unregisterClient(c *client) {
	p.clientsMu.Lock()
	delete(p.clients, c)
	p.clientsMu.Unlock()
}

// disconnectClients finalizes every connected client (stage two of client
// shutdown). It snapshots the set under the lock and calls disconnect outside
// it, because client.disconnect calls back into unregisterClient (which takes
// the same lock). disconnect is idempotent, so a client already finalized is
// unaffected.
func (p *Pipeline) disconnectClients() {
	p.clientsMu.Lock()
	clients := make([]*client, 0, len(p.clients))
	for c := range p.clients {
		clients = append(clients, c)
	}
	p.clientsMu.Unlock()

	for _, c := range clients {
		c.disconnect()
	}
}

// startReaper initializes the reaper state and launches the reaper goroutine.
// Called once from each pipeline constructor.
func (p *Pipeline) startReaper() {
	p.reaperPending = make(map[*client]struct{})
	p.reaperNotify = make(chan struct{}, 1)
	p.reaperDone = make(chan struct{})
	p.reaperWG.Go(func() {
		p.reapClosedClients()
	})
}

// finalizeWhenDrained hands a Closed client to the reaper so it is finalized
// (stage two) as soon as its already-published events are acknowledged, rather
// than lingering until the whole pipeline disconnects.
func (p *Pipeline) finalizeWhenDrained(c *client) {
	p.reaperMu.Lock()
	p.reaperPending[c] = struct{}{}
	p.reaperMu.Unlock()
	// Wake the reaper so it rebuilds its wait set. Non-blocking: a pending
	// notify already covers this change.
	select {
	case p.reaperNotify <- struct{}{}:
	default:
	}
}

// reapClosedClients runs as a single goroutine for the pipeline's lifetime. It
// finalizes Closed-but-not-yet-drained clients as their events are
// acknowledged. Each pass non-blockingly sweeps the pending set and finalizes
// every client whose ACKWaitChan has closed — O(pending) per pass, so a burst
// of closing clients drains in one pass rather than the O(N^2) a per-client
// wait would cost. When nothing is pending it blocks until a client is handed
// over or the pipeline disconnects; otherwise it re-sweeps every reaperInterval.
// Using one goroutine (not one per client) also keeps it out of per-client
// goroutine-leak accounting.
func (p *Pipeline) reapClosedClients() {
	for {
		p.reaperMu.Lock()
		var ready []*client
		for c := range p.reaperPending {
			select {
			case <-c.producer.ACKWaitChan():
				ready = append(ready, c)
			default:
			}
		}
		for _, c := range ready {
			delete(p.reaperPending, c)
		}
		pending := len(p.reaperPending)
		p.reaperMu.Unlock()

		for _, c := range ready {
			c.disconnect()
		}

		if pending == 0 {
			// Nothing to watch: block until a client is handed over or we stop.
			select {
			case <-p.reaperDone:
				return
			case <-p.reaperNotify:
			}
		} else {
			// Some clients are still draining: re-sweep soon.
			select {
			case <-p.reaperDone:
				return
			case <-p.reaperNotify:
			case <-time.After(reaperInterval):
			}
		}
	}
}

// Connect creates a new client with default settings.
func (p *Pipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

// ConnectWith create a new Client for publishing events to the pipeline.
// The client behavior on close and ACK handling can be configured by setting
// the appropriate fields in provided ClientConfig.
// If not set otherwise the default publish mode is OutputChooses.
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

	// Note: cfg.WaitClose no longer makes client.Close block. Pipeline.Disconnect
	// (bounded by its context) is now responsible for waiting on outstanding
	// acknowledgments — see issues #50104 and #49794.

	processors, err := p.createEventProcessing(cfg.Processing, publishDisabled)
	if err != nil {
		return nil, err
	}

	clientListener := cfg.ClientListener
	if clientListener == nil {
		clientListener = noopClientListener{}
	}

	client := &client{
		logger:         p.monitors.Logger,
		clientListener: clientListener,
		processors:     processors,
		eventFlags:     eventFlags,
		canDrop:        canDrop,
		observer:       p.observer,
	}

	client.isOpen.Store(true)

	ackHandler := cfg.EventListener

	producerCfg := queue.ProducerConfig{
		ACK: func(count int) {
			client.observer.eventsACKed(count)
			if ackHandler != nil {
				ackHandler.ACKEvents(count)
			}
		},
	}

	if ackHandler == nil {
		ackHandler = acker.Nil()
	}

	client.eventListener = ackHandler
	client.producer = p.outputController.queueProducer(producerCfg)
	if client.producer == nil {
		// This can only happen if the pipeline was shut down while clients
		// were still waiting to connect.
		return nil, fmt.Errorf("client failed to connect because the pipeline is shutting down")
	}

	// Register the client so the Pipeline can finalize it (stage two of
	// shutdown) when the pipeline disconnects. The client removes itself from
	// the registry when it is disconnected, and hands itself to the reaper on
	// Close so it is finalized as soon as its events drain.
	client.onRemove = func() { p.unregisterClient(client) }
	client.requestFinalize = func() { p.finalizeWhenDrained(client) }
	p.registerClient(client)

	p.observer.clientConnected()
	return client, nil
}

func (p *Pipeline) createEventProcessing(cfg beat.ProcessingConfig, noPublish bool) (beat.Processor, error) {
	if p.processors == nil {
		return nil, nil
	}
	return p.processors.Create(cfg, noPublish)
}

// OutputReloader returns a reloadable object for the output section of this pipeline
func (p *Pipeline) OutputReloader() OutputReloader {
	if r, ok := p.outputController.(OutputReloader); ok {
		return r
	}
	return noopReloader{}
}

// Parses the given config and returns a QueueFactory based on it.
// This helper exists to frontload config parsing errors: if there is an
// error in the queue config, we want it to show up as fatal during
// initialization, even if the queue itself isn't created until later.
// It also returns the parsed queue settings (with defaults applied) so callers
// can detect mismatched configs between pipelines that connect with the same
// shared intake queue id.
func queueFactoryForUserConfig(queueType string, userConfig *conf.C, paths *paths.Path) (queue.QueueFactory[publisher.Event], any, error) {
	switch queueType {
	case memqueue.QueueType:
		settings, err := memqueue.SettingsForUserConfig(userConfig)
		if err != nil {
			return nil, nil, err
		}
		return memqueue.FactoryForSettings[publisher.Event](settings), settings, nil
	case slabqueue.QueueType:
		settings, err := slabqueue.SettingsForUserConfig(userConfig)
		if err != nil {
			return nil, nil, err
		}
		return slabqueue.FactoryForSettings[publisher.Event](settings), settings, nil
	case diskqueue.QueueType:
		settings, err := diskqueue.SettingsForUserConfig(userConfig)
		if err != nil {
			return nil, nil, err
		}
		return diskqueue.FactoryForSettings(settings, paths), settings, nil
	default:
		return nil, nil, fmt.Errorf("unrecognized queue type '%v'", queueType)
	}
}

type noopReloader struct{}

func (n noopReloader) Reload(
	cfg *reload.ConfigWithMeta,
	_ func(outputs.Observer, conf.Namespace) (outputs.Group, error),
) error {
	// This function should never be called, but if it is, return an error we can troubleshoot.
	var unitID string
	if cfg != nil {
		unitID = cfg.InputUnitID
	}
	return fmt.Errorf("unsupported reload triggered by unit '%v'", unitID)
}

type noopClientListener struct{}

func (n noopClientListener) Closing()                    {}
func (n noopClientListener) Closed()                     {}
func (n noopClientListener) NewEvent()                   {}
func (n noopClientListener) Filtered()                   {}
func (n noopClientListener) Published()                  {}
func (n noopClientListener) DroppedOnPublish(beat.Event) {}
