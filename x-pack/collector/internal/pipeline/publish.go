package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/x-pack/collector/internal/publishing"
	"github.com/elastic/go-concert/unison"
)

// The processing pipeline disables all default fields on purpose. Defaults should always be added
// explicitely by agent/fleet using add_X_metadata or add_fields processor.
//
// NOTE: Global Beats settings like processor, name, fields, tags and timeseries.enabled will be ignored.
var processorsSupport = processing.MakeDefaultSupport(true)
var emptyConfig = common.NewConfig()

var errEventFlitered = errors.New("event filtered out")

// TODO: all/most of this to the publishing package?
type pipelinePublishing struct {
	log              *logp.Logger
	processorFactory processing.Supporter
	events           *eventTracker
	outputMigrations managedGroup
	output           *replacableOutput
}

type client struct {
	mu      sync.Mutex
	closeWG unison.SafeWaitGroup

	log  *logp.Logger
	mode beat.PublishMode

	events    *eventTracker
	eventsCtx *eventContext

	output    *replacableOutput
	processor beat.Processor
	acker     beat.ACKer
}

func createPublishPipeline(log *logp.Logger, info beat.Info, output output) (*pipelinePublishing, error) {
	processorFactory, err := processorsSupport(info, log, emptyConfig)
	if err != nil {
		// We configure the processor using a static config. This must never fail
		panic(err)
	}

	events := &eventTracker{}

	pipelineOutput, err := newPipelineOutput(log, 0, output, events)
	if err != nil {
		return nil, err
	}

	return &pipelinePublishing{
		log:              log,
		processorFactory: processorFactory,
		events:           events,
		output:           newReplacablePublisher(pipelineOutput),
	}, nil
}

func (p *pipelinePublishing) Close() error {
	p.outputMigrations.Stop()
	err := p.output.Close()
	p.output = nil
	return err
}

func (p *pipelinePublishing) UpdateOutput(out output) error {
	// no change, no work
	old := p.output.GetActive()
	if old.output.configHash == out.configHash {
		return nil
	}

	outputID := old.id + 1
	replacement, err := newPipelineOutput(p.log, outputID, out, p.events)
	if err != nil {
		return err
	}

	// atomically update the outputID for all ACK handlers, such that ACKs from the currently active output
	// are ignored. There is still the chance that clients send new events to the old output, but
	// we clean this up when we send old non-acked events to the new output.
	// The number of events that get ACKed can not change anymore after this block has succeeded.
	old.acker.closed.Store(true)
	p.events.SetOutputID(outputID)

	// Block all clients while we try to make a snapshot of the current state.
	// We want to send again all events that are still pending, such that we can
	// correctly receive ACK from the new output. In order to not get an event ACKed
	// twice we need to ensure that clients can not publish new events while we swap out the output.
	//
	// IMPORTANT: Making the snapshot of pending events and replacing the output
	//            must be atomic, such that we do not ignore any events, that would otherwise
	//            be send to the old output by accident.
	//
	// IMPORTANT: Clients must be able to shutdown without waiting for too long. All works within this
	//            critical section must not block unbounded. It must only be
	//            required to collect information that is required to fixup
	//            processing after the output swap. No IO or send via channels must happen here.
	p.events.Lock()
	p.output.SetActive(replacement)
	snapshot := p.events.SnapshotPending()
	p.events.Unlock()

	oldMigrations := p.outputMigrations.FindAll(func(_ string) bool { return true })
	cancelAll(oldMigrations)
	old.Close()

	if len(snapshot) > 0 {
		p.outputMigrations.Go(out.configHash, func(cancel unison.Canceler) {
			for ; len(snapshot) > 0 && cancel.Err() == nil; snapshot = snapshot[1:] {
				pending := snapshot[0]
				err := replacement.Publish(pending.mode, pending.id, pending.event)
				if err != nil {
					replacement.acker.UpdateEventStatus(pending.id, publishing.EventFailed)
				}
			}
			return
		})
	}

	waitAll(oldMigrations)
	return nil
}

func (p *pipelinePublishing) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *pipelinePublishing) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	if err := validateClientConfig(&cfg); err != nil {
		return nil, err
	}

	processors, err := p.processorFactory.Create(cfg.Processing, false)
	if err != nil {
		return nil, err
	}
	cfg.Processing = beat.ProcessingConfig{
		DisableHost: cfg.Processing.DisableHost,
		KeepNull:    cfg.Processing.KeepNull,
		Private:     cfg.Processing.Private,
	}

	acker := cfg.ACKHandler
	return &client{
		log:  p.log,
		mode: cfg.PublishMode,

		events:    p.events,
		eventsCtx: p.events.Register(cfg.PublishMode, acker),
		output:    p.output,

		processor: processors,
		acker:     cfg.ACKHandler,
	}, nil
}

func (c *client) Close() error {
	c.closeWG.Close()
	c.events.Unregister(c.eventsCtx)
	c.closeWG.Wait()
	return nil
}

func (c *client) PublishAll(events []beat.Event) {
	if err := c.closeWG.Add(1); err != nil {
		return
	}
	defer c.closeWG.Done()

	for _, event := range events {
		c.publishEvent(event)
	}
}

func (c *client) Publish(event beat.Event) {
	if err := c.closeWG.Add(1); err != nil {
		return
	}
	defer c.closeWG.Done()

	c.publishEvent(event)
}

func (c *client) publishEvent(event beat.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var filteredEvent *beat.Event
	if c.processor != nil {
		var err error
		filteredEvent, err = c.processor.Run(&event)
		if err != nil {
			c.log.Errorf("Processing failed: %v", err)
		}
	} else {
		filteredEvent = &event
	}

	if filteredEvent == nil {
		if c.acker != nil {
			c.acker.AddEvent(event, false)
		}
		return errEventFlitered
	}

	if c.acker != nil {
		c.acker.AddEvent(event, true)
	}

	id := c.eventsCtx.RecordEvent(event)
	activeOutput := c.output.GetActive()
	if activeOutput == nil {
		return nil // event is recorded. Publishing gets postponed until after new output is in place
	}

	err := activeOutput.Publish(c.mode, id, event)
	if err != nil && err != context.Canceled {
		activeOutput.acker.UpdateEventStatus(id, publishing.EventFailed)
	}
	return err
}

// validateClientConfig checks a ClientConfig can be used with (*Pipeline).ConnectWith.
func validateClientConfig(c *beat.ClientConfig) error {
	withDrop := false

	switch m := c.PublishMode; m {
	case beat.DefaultGuarantees, beat.GuaranteedSend, beat.OutputChooses:
	case beat.DropIfFull:
		withDrop = true
	default:
		return fmt.Errorf("unknown publish mode %v", m)
	}

	// ACK handlers can not be registered DropIfFull is set, as dropping events
	// due to full broker can not be accounted for in the clients acker.
	if c.ACKHandler != nil && withDrop {
		return errors.New("ACK handlers with DropIfFull mode not supported")
	}

	return nil
}
