// Package pipeline combines all publisher functionality (processors, broker,
// outputs) to create instances of complete publisher pipelines, beats can
// connect to publish events to.
package pipeline

import (
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/beat"
	"github.com/elastic/beats/libbeat/publisher/broker"
)

// Pipeline implementation providint all beats publisher functionality.
// The pipeline consists of clients, processors, a central broker, an output
// controller and the actual outputs.
// The broker implementing the broker.Broker interface is the most entral entity
// to the pipeline, providing support for pushung, batching and pulling events.
// The pipeline adds different ACKing strategies and wait close support on top
// of the broker. For handling ACKs, the pipeline keeps track of filtered out events,
// to be ACKed to the client in correct order.
// The output controller configures a (potentially reloadable) set of load
// balanced output clients. Events will be pulled from the broker and pushed to
// the output clients using a shared work queue for the active outputs.Group.
// Processors in the pipeline are executed in the clients go-routine, before
// entering the broker. No filtering/processing will occur on the output side.
type Pipeline struct {
	logger *logp.Logger
	broker broker.Broker
	output *outputController

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

	processors pipelineProcessors
}

type pipelineProcessors struct {
	// The pipeline its processor settings for
	// constructing the clients complete processor
	// pipeline on connect.
	beatMetaProcessor  beat.Processor
	eventMetaProcessor beat.Processor
	processors         beat.Processor
	disabled           bool // disabled is set if outputs have been disabled via CLI
}

// Settings is used to pass additional settings to a newly created pipeline instance.
type Settings struct {
	// WaitClose sets the maximum duration to block when clients or pipeline itself is closed.
	// When and how WaitClose is applied depends on WaitCloseMode.
	WaitClose time.Duration

	WaitCloseMode WaitCloseMode

	Annotations Annotations
	Processors  *processors.Processors

	Disabled bool
}

// Annotations configures additional metadata to be adde to every single event
// being published. The meta data will be added before executing the configured
// processors, so all processors configured with the pipeline or client will see
// the same/complete event.
type Annotations struct {
	Beat  common.MapStr
	Event common.EventMetadata
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

type pipelineEventer struct {
	mutex      sync.Mutex
	modifyable bool

	waitClose *waitCloser
	cb        *pipelineEventCB
}

type waitCloser struct {
	// keep track of total number of active events (minus dropped by processors)
	events sync.WaitGroup
}

type brokerFactory func(broker.Eventer) (broker.Broker, error)

// Load uses a Config object to create a new complete Pipeline instance with
// configured broker and outputs.
func Load(beatInfo common.BeatInfo, config Config) (*Pipeline, error) {
	if !config.Output.IsSet() {
		return nil, errors.New("no output configured")
	}

	brokerFactory := func(e broker.Eventer) (broker.Broker, error) {
		return broker.Load(e, config.Broker)
	}

	output, err := outputs.Load(beatInfo, config.Output.Name(), config.Output.Config())
	if err != nil {
		return nil, err
	}

	// TODO: configure pipeline processors
	pipeline, err := New(brokerFactory, output, Settings{
		WaitClose:     config.WaitShutdown,
		WaitCloseMode: WaitOnPipelineClose,
	})
	if err != nil {
		for _, c := range output.Clients {
			c.Close()
		}
		return nil, err
	}

	return pipeline, nil
}

// New create a new Pipeline instance from a broker instance and a set of outputs.
// The new pipeline will take ownership of broker and outputs. On Close, the
// broker and outputs will be closed.
func New(
	brokerFactory brokerFactory,
	out outputs.Group,
	settings Settings,
) (*Pipeline, error) {

	annotations := settings.Annotations
	var err error

	var beatMeta beat.Processor
	if meta := annotations.Beat; meta != nil {
		beatMeta = beatAnnotateProcessor(meta)
	}

	var eventMeta beat.Processor
	if em := annotations.Event; len(em.Fields) > 0 || len(em.Tags) > 0 {
		eventMeta = eventAnnotateProcessor(em)
	}

	var prog beat.Processor
	if ps := settings.Processors; ps != nil && len(ps.List) > 0 {
		tmp := &program{title: "global"}
		for _, p := range ps.List {
			tmp.add(p)
		}
		prog = tmp
	}

	log := defaultLogger
	p := &Pipeline{
		logger:           log,
		waitCloseMode:    settings.WaitCloseMode,
		waitCloseTimeout: settings.WaitClose,
		processors: pipelineProcessors{
			beatMetaProcessor:  beatMeta,
			eventMetaProcessor: eventMeta,
			processors:         prog,
			disabled:           settings.Disabled,
		},
	}
	p.ackBuilder = &pipelineEmptyACK{p}
	p.ackActive = atomic.MakeBool(true)
	p.eventer.modifyable = true

	if settings.WaitCloseMode == WaitOnPipelineClose && settings.WaitClose > 0 {
		p.waitCloser = &waitCloser{}

		// waitCloser decrements counter on broker ACK (not per client)
		p.eventer.waitClose = p.waitCloser
	}

	p.broker, err = brokerFactory(&p.eventer)
	if err != nil {
		return nil, err
	}
	p.eventSema = newSema(p.broker.BufferConfig().Events)

	p.output = newOutputController(log, p.broker)
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

// Close stops the pipeline, outputs and broker.
// If WaitClose with WaitOnPipelineClose mode is configured, Close will block
// for a duration of WaitClose, if there are still active events in the pipeline.
// Note: clients must be closed before calling Close.
func (p *Pipeline) Close() error {
	log := p.logger

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

	// close output before shutting down broker
	p.output.Close()

	// shutdown broker
	err := p.broker.Close()
	if err != nil {
		log.Err("pipeline broker shutdown error: ", err)
	}

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

	processors := p.newProcessorPipeline(cfg)

	acker := p.makeACKer(processors != nil, &cfg, waitClose)
	producerCfg := broker.ProducerConfig{
		// only cancel events from broker if acker is configured
		// and no pipeline-wide ACK handler is registered
		DropOnCancel: acker != nil && p.eventer.cb == nil,
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

	producer := p.broker.Producer(producerCfg)
	client := &client{
		pipeline:     p,
		eventer:      cfg.Events,
		processors:   processors,
		producer:     producer,
		acker:        acker,
		eventFlags:   eventFlags,
		canDrop:      canDrop,
		reportEvents: reportEvents,
	}

	return client, nil
}

func (e *pipelineEventer) OnACK(n int) {
	if wc := e.waitClose; wc != nil {
		wc.dec(n)
	}
	if e.cb != nil {
		e.cb.reportBrokerACK(n)
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
