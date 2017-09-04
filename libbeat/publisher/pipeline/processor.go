package pipeline

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/processors"
)

type program struct {
	title string
	list  []beat.Processor
}

type processorFn struct {
	name string
	fn   func(event *beat.Event) (*beat.Event, error)
}

// newProcessorPipeline prepares the processor pipeline, merging
// post processing, event annotations and actual configured processors.
// The pipeline generated ensure the client and pipeline processors
// will see the complete events with all meta data applied.
//
// Pipeline (C=client, P=pipeline)
//
//  1. (P) generalize/normalize event
//  2. (C) add Meta from client Config to event.Meta
//  3. (C) add Fields from client config to event.Fields
//  4. (C) add client fields + tags
//  5. (C) client processors list
//  6. (P) add beats metadata
//  7. (P) add pipeline fields + tags
//  8. (P) pipeline processors list
//  9. (P) (if publish/debug enabled) log event
// 10. (P) (if output disabled) dropEvent
func (p *Pipeline) newProcessorPipeline(
	config beat.ClientConfig,
) beat.Processor {
	var (
		// pipeline processors
		processors = &program{title: "processPipeline"}

		// client fields and metadata
		clientMeta      = config.Meta
		clientFields    = buildFields(config.Fields, config.EventMetadata)
		clientTags      = config.EventMetadata.Tags
		localProcessors = makeClientProcessors(config)

		// pipeline global
		global          = p.processors
		fieldsProcessor = global.fieldsProcessor
		tagsProcessor   = global.tagsProcessor
	)

	// setup 1: generalize/normalize output (P)
	processors.add(generalizeProcessor)

	// setup 2: add Meta from client config (C)
	if m := clientMeta; len(m) > 0 {
		processors.add(clientEventMeta(m))
	}

	if localProcessors == nil {
		// merge 3 + 4 + 5 + 7:
		//   Merge all field and  updates into one processor if no client
		//   processors have been configured.
		fields := clientFields.Clone()
		fields.DeepUpdate(global.fields)
		fieldsProcessor = makeAddFieldsProcessor("fields", fields)

		if tags := append(clientTags, global.tags...); len(tags) > 0 {
			tagsProcessor = makeAddTagsProcessor("tags", tags)
		}
	} else {
		// setup 3 + 4: add Fields from client config (C), add fields + tags (C)
		if f := clientFields; len(f) > 0 {
			processors.add(makeAddFieldsProcessor("clientFields", f))
		}
		if t := clientTags; len(t) > 0 {
			processors.add(makeAddTagsProcessor("clientTags", t))
		}

		// setup 5: client processors (C)
		processors.add(localProcessors)
	}

	// setup 6 + 7: add beats metadata (P), add event fields + tags (P)
	processors.add(fieldsProcessor)
	processors.add(tagsProcessor)

	// setup 8: pipeline processors (P)
	processors.add(global.processors)

	// setup 9: debug print final event (P)
	if logp.IsDebug("publish") {
		processors.add(debugPrintProcessor())
	}

	// setup 10: drop all events if outputs are disabled (P)
	if global.disabled {
		processors.add(dropDisabledProcessor)
	}

	return processors
}

func (p *program) add(processor processors.Processor) {
	if processor != nil {
		p.list = append(p.list, processor)
	}
}

func (p *program) String() string {
	var s []string
	for _, p := range p.list {
		s = append(s, p.String())
	}

	str := strings.Join(s, ", ")
	if p.title == "" {
		return str
	}
	return fmt.Sprintf("%v{%v}", p.title, str)
}

func (p *program) Run(event *beat.Event) (*beat.Event, error) {
	if p == nil || len(p.list) == 0 {
		return event, nil
	}

	for _, sub := range p.list {
		var err error

		event, err = sub.Run(event)
		if err != nil {
			// XXX: We don't drop the event, but continue filtering here iff the most
			//      recent processor did return an event.
			//      We want processors having this kind of implicit behavior
			//      on errors?

			logp.Debug("filter", "fail to apply processor %s: %s", p, err)
		}

		if event == nil {
			return nil, err
		}
	}

	return event, nil
}

func newProcessor(name string, fn func(*beat.Event) (*beat.Event, error)) *processorFn {
	return &processorFn{name: name, fn: fn}
}

func newAnnotateProcessor(name string, fn func(*beat.Event)) *processorFn {
	return newProcessor(name, func(event *beat.Event) (*beat.Event, error) {
		fn(event)
		return event, nil
	})
}

func (p *processorFn) String() string                         { return p.name }
func (p *processorFn) Run(e *beat.Event) (*beat.Event, error) { return p.fn(e) }

var generalizeProcessor = newProcessor("generalizeEvent", func(event *beat.Event) (*beat.Event, error) {

	// Filter out empty events. Empty events are still reported by ACK callbacks.
	if len(event.Fields) == 0 {
		return nil, nil
	}

	fields := common.ConvertToGenericEvent(event.Fields)
	if fields == nil {
		logp.Err("fail to convert to generic event")
		return nil, nil
	}

	event.Fields = fields
	return event, nil
})

var dropDisabledProcessor = newProcessor("dropDisabled", func(event *beat.Event) (*beat.Event, error) {
	return nil, nil
})

func beatAnnotateProcessor(beatMeta common.MapStr) *processorFn {
	const key = "beat"
	return newAnnotateProcessor("annotateBeat", func(event *beat.Event) {
		if orig, exists := event.Fields["beat"]; !exists {
			event.Fields[key] = beatMeta.Clone()
		} else if M, ok := orig.(common.MapStr); !ok {
			event.Fields[key] = beatMeta.Clone()
		} else {
			event.Fields[key] = common.MapStrUnion(beatMeta, M)
		}
	})
}

func eventAnnotateProcessor(eventMeta common.EventMetadata) *processorFn {
	return newAnnotateProcessor("annotateEvent", func(event *beat.Event) {
		common.AddTags(event.Fields, eventMeta.Tags)
		if fields := eventMeta.Fields; len(fields) > 0 {
			common.MergeFields(event.Fields, fields.Clone(), eventMeta.FieldsUnderRoot)
		}
	})
}

func clientEventMeta(meta common.MapStr) *processorFn {
	return newAnnotateProcessor("@metadata", func(event *beat.Event) {
		if event.Meta == nil {
			event.Meta = meta.Clone()
		} else {
			event.Meta = event.Meta.Clone()
			event.Meta.DeepUpdate(meta.Clone())
		}
	})
}

func pipelineEventFields(fields common.MapStr) *processorFn {
	return makeAddFieldsProcessor("pipelineFields", fields)
}

func makeAddTagsProcessor(name string, tags []string) *processorFn {
	return newAnnotateProcessor(name, func(event *beat.Event) {
		common.AddTags(event.Fields, tags)
	})
}

func makeAddFieldsProcessor(name string, fields common.MapStr) *processorFn {
	return newAnnotateProcessor(name, func(event *beat.Event) {
		event.Fields.DeepUpdate(fields.Clone())
	})
}

func debugPrintProcessor() *processorFn {
	// ensure only one go-routine is using the encoder (in case
	// beat.Client is shared between multiple go-routines by accident)
	var mux sync.Mutex

	encoder := json.New(true)
	return newProcessor("debugPrint", func(event *beat.Event) (*beat.Event, error) {
		mux.Lock()
		defer mux.Unlock()

		b, err := encoder.Encode("<not set>", event)
		if err != nil {
			return event, nil
		}

		logp.Debug("publish", "Publish event: %s", b)
		return event, nil
	})
}

func makeClientProcessors(config beat.ClientConfig) processors.Processor {
	procs := config.Processor
	if procs == nil || len(procs.All()) == 0 {
		return nil
	}

	return &program{
		title: "client",
		list:  procs.All(),
	}
}

func mergeFields(a, b common.MapStr) common.MapStr {
	m := a.Clone()
	m.DeepUpdate(b)
	return m
}

func buildFields(fields common.MapStr, em common.EventMetadata) common.MapStr {
	if fields == nil {
		fields = common.MapStr{}
	} else {
		fields = fields.Clone()
	}

	if fs := em.Fields; len(fs) > 0 {
		common.MergeFields(fields, fs.Clone(), em.FieldsUnderRoot)
	}

	if len(fields) == 0 {
		return nil
	}
	return fields
}
