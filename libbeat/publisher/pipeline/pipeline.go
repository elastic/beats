// Package pipeline combines all publisher functionality (processors, broker,
// outputs) to create instances of complete publisher pipelines, beats can
// connect to publish events to.
package pipeline

import (
	"errors"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
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

	waitClose     time.Duration
	waitCloseMode WaitCloseMode

	// keep track of total number of active events (minus dropped by processors)
	events sync.WaitGroup

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

// Load uses a Config object to create a new complete Pipeline instance with
// configured broker and outputs.
func Load(beatInfo common.BeatInfo, config Config) (*Pipeline, error) {
	if !config.Output.IsSet() {
		return nil, errors.New("no output configured")
	}

	broker, err := broker.Load(config.Broker)
	if err != nil {
		return nil, err
	}

	output, err := outputs.Load(beatInfo, config.Output.Name(), config.Output.Config())
	if err != nil {
		broker.Close()
		return nil, err
	}

	// TODO: configure pipeline processors
	pipeline, err := New(broker, output, Settings{
		WaitClose:     config.WaitShutdown,
		WaitCloseMode: WaitOnPipelineClose,
	})
	if err != nil {
		broker.Close()
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
	broker broker.Broker,
	out outputs.Group,
	settings Settings,
) (*Pipeline, error) {

	annotations := settings.Annotations

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
		logger:             log,
		broker:             broker,
		output:             newOutputController(log, broker),
		waitClose:          settings.WaitClose,
		waitCloseMode:      settings.WaitCloseMode,
		beatMetaProcessor:  beatMeta,
		eventMetaProcessor: eventMeta,
		processors:         prog,
		disabled:           settings.Disabled,
	}

	p.output.Set(out)
	return p, nil
}

// Close stops the pipeline, outputs and broker.
// If WaitClose with WaitOnPipelineClose mode is configured, Close will block
// for a duration of WaitClose, if there are still active events in the pipeline.
// Note: clients must be closed before calling Close.
func (p *Pipeline) Close() error {
	log := p.logger

	log.Debug("close pipeline")

	if p.waitClose > 0 && p.waitCloseMode == WaitOnPipelineClose {
		ch := make(chan struct{})
		go func() {
			p.events.Wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			// all events have been ACKed

		case <-time.After(p.waitClose):
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

func (p *Pipeline) activeEventsAdd(n int) {
	p.events.Add(n)
}

func (p *Pipeline) activeEventsDone(n int) {
	for i := 0; i < n; i++ {
		p.events.Done()
	}
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

	switch cfg.PublishMode {
	case beat.GuaranteedSend:
		eventFlags = publisher.GuaranteedSend
	case beat.DropIfFull:
		canDrop = true
	}

	waitClose := cfg.WaitClose
	var reportEvents bool

	switch p.waitCloseMode {
	case NoWaitOnClose:

	case WaitOnClientClose:
		if waitClose <= 0 {
			waitClose = p.waitClose
		}

	case WaitOnPipelineClose:
		reportEvents = p.waitClose > 0
	}

	processors := p.newProcessorPipeline(cfg)
	acker := makeACKer(processors != nil, &cfg, waitClose)
	producerCfg := broker.ProducerConfig{}

	// only cancel events from broker if acker is configured
	cancelEvents := acker != nil

	// configure client and acker to report events to pipeline.events
	// for handling waitClose
	if reportEvents {
		if acker == nil {
			acker = nilACKer
		}

		acker = &pipelineACK{
			pipeline: p,
			acker:    acker,
		}
		producerCfg.OnDrop = p.activeEventsDone
	}

	if acker != nil {
		producerCfg.ACK = acker.ackEvents
	} else {
		acker = nilACKer
	}

	producer := p.broker.Producer(producerCfg)
	client := &client{
		pipeline:     p,
		processors:   processors,
		producer:     producer,
		acker:        acker,
		eventFlags:   eventFlags,
		canDrop:      canDrop,
		cancelEvents: cancelEvents,
		reportEvents: reportEvents,
	}

	return client, nil
}

func makeACKer(
	withProcessors bool,
	cfg *beat.ClientConfig,
	waitClose time.Duration,
) acker {
	// maximum number of events that can be published (including drops) without ACK.
	//
	// TODO: this MUST be configurable and should be max broker buffer size...
	gapEventBuffer := 64

	switch {
	case cfg.ACKCount != nil:
		return makeCountACK(withProcessors, gapEventBuffer, waitClose, cfg.ACKCount)
	case cfg.ACKEvents != nil:
		return newEventACK(withProcessors, gapEventBuffer, waitClose, cfg.ACKEvents)
	case cfg.ACKLastEvent != nil:
		return newEventACK(withProcessors, gapEventBuffer, waitClose, lastEventACK(cfg.ACKLastEvent))
	}
	return nil
}
