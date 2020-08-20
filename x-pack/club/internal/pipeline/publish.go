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
	"github.com/elastic/beats/v7/x-pack/club/internal/publishing"
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
	out              publishing.Publisher
	events           *eventTracker
}

type output struct {
	configHash string
	output     publishing.Output
}

type client struct {
	mu      sync.Mutex
	closeWG unison.SafeWaitGroup

	log  *logp.Logger
	mode beat.PublishMode

	events    *eventTracker
	eventsCtx *eventContext

	output    publishing.Publisher
	processor beat.Processor
	acker     beat.ACKer
}

type eventTracker struct {
	// contexts stores the status of events. Each client has it's own status. This guarantees that we can
	// correctly ACK events to the original publisher.
	// The freelist tracks event contexts currently not in use. Old contexts get
	// reuse when a new client is connected (ensure IDs are stable and lookup is O(1)
	// Modifications to contextsMu and freelist must be protected using contextsMu.
	contextsMu sync.Mutex
	contexts   []*eventContext
	freelist   []uint32
}

type eventContext struct {
	ref uint32

	contextID uint32

	acker beat.ACKer

	statusMu sync.Mutex
	startID  uint32

	// TODO: track event memory usage, so we account for overal memory usage for
	// events in progress and events published
	status []publishing.EventStatus
}

func createPublishPipeline(log *logp.Logger, info beat.Info, output output) (*pipelinePublishing, error) {
	processorFactory, err := processorsSupport(info, log, emptyConfig)
	if err != nil {
		// We configure the processor using a static config. This must never fail
		panic(err)
	}

	events := &eventTracker{}

	out, err := output.output.Open(context.Background(), log, events)
	if err != nil {
		return nil, err
	}

	return &pipelinePublishing{
		log:              log,
		out:              out,
		processorFactory: processorFactory,
		events:           events,
	}, nil
}

func (p *pipelinePublishing) Close() error {
	err := p.out.Close()
	p.out = nil
	return err
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
		eventsCtx: p.events.Register(acker),
		output:    p.out,

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

	id := c.eventsCtx.RecordEvent()
	err := c.output.Publish(c.mode, id, event)
	if err != nil && err != context.Canceled {
		c.events.UpdateEventStatus(id, publishing.EventFailed)
	}
	return err
}

func (t *eventTracker) Register(acker beat.ACKer) *eventContext {
	t.contextsMu.Lock()
	defer t.contextsMu.Unlock()

	var idx uint32
	if L := len(t.freelist); L > 0 {
		idx = uint32(L - 1)
		t.freelist = t.freelist[:L-1]
	} else {
		idx = uint32(len(t.contexts))
		t.contexts = append(t.contexts, nil)
	}

	ectx := &eventContext{
		ref:       1,
		contextID: idx,
		acker:     acker,
	}

	t.contexts[idx] = ectx
	return ectx
}

func (t *eventTracker) Unregister(ctx *eventContext) {
	ctx.statusMu.Lock()
	ctx.ref--
	release := ctx.ref == 0
	ctx.statusMu.Unlock()

	if release {
		t.release(ctx)
	}
}

func (t *eventTracker) release(ctx *eventContext) {
	t.contextsMu.Lock()
	defer t.contextsMu.Unlock()

	idx := ctx.contextID
	t.contexts[idx] = nil

	// TODO: try to free more space
	if L := len(t.contexts); L-1 == int(idx) {
		t.contexts = t.contexts[:L-1]
	} else {
		t.freelist = append(t.freelist, idx)
	}

}

func (t *eventTracker) UpdateEventStatus(id publishing.EventID, status publishing.EventStatus) {
	idx := id >> 32
	t.contextsMu.Lock()
	ec := t.contexts[idx]
	t.contextsMu.Unlock()

	more := ec.updateStatus(id, status)
	if !more {
		t.release(ec)
	}
}

func (c *eventContext) RecordEvent() publishing.EventID {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()

	idx := len(c.status)
	c.status = append(c.status, publishing.EventPending)
	c.ref++

	id := publishing.EventID(uint64(c.contextID)<<32 | uint64(c.startID+uint32(idx)))
	return id
}

func (c *eventContext) updateStatus(id publishing.EventID, status publishing.EventStatus) bool {
	refs, acked := func() (uint32, uint32) {
		c.statusMu.Lock()
		defer c.statusMu.Unlock()

		idx := uint32(id) - c.startID
		c.status[idx] = status

		var n uint32
		for _, st := range c.status {
			if st == publishing.EventPending {
				break
			}
			n++
		}

		c.startID += n
		c.status = c.status[n:]
		c.ref -= n
		return c.ref, n
	}()

	if acked > 0 && c.acker != nil {
		c.acker.ACKEvents(int(acked))
	}

	return refs > 0
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
