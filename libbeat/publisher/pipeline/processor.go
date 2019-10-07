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
//  4. (P) add pipeline fields + tags
//  5. (C) add client fields + tags
//  6. (C) client processors list
//  7. (P) add beats metadata
//  8. (P) pipeline processors list
//  9. (P) (if publish/debug enabled) log event
// 10. (P) (if output disabled) dropEvent
func newProcessorPipeline(
	info beat.Info,
	global pipelineProcessors,
	config beat.ClientConfig,
) beat.Processor {
	var (
		// pipeline processors
		processors = &program{title: "processPipeline"}

		// client fields and metadata
		clientMeta      = config.Meta
		localProcessors = makeClientProcessors(config)
	)

	needsCopy := global.alwaysCopy || localProcessors != nil || global.processors != nil
	builtin := global.builtinMeta

	if !config.SkipNormalization {
		// setup 1: generalize/normalize output (P)
		processors.add(generalizeProcessor)
	}

	// setup 2: add Meta from client config (C)
	if m := clientMeta; len(m) > 0 {
		processors.add(clientEventMeta(m, needsCopy))
	}

	// setup 4, 5: pipeline tags + client tags
	var tags []string
	tags = append(tags, global.tags...)
	tags = append(tags, config.EventMetadata.Tags...)
	if len(tags) > 0 {
		processors.add(makeAddTagsProcessor("tags", tags))
	}

	// setup 3, 4, 5: client config fields + pipeline fields + client fields + dyn metadata
	fields := config.Fields.Clone()
	fields.DeepUpdate(global.fields.Clone())
	if em := config.EventMetadata; len(em.Fields) > 0 {
		common.MergeFieldsDeep(fields, em.Fields.Clone(), em.FieldsUnderRoot)
	}

	if len(fields) > 0 {
		// Enforce a copy of fields if dynamic fields are configured or beats
		// metadata will be merged into the fields.
		// With dynamic fields potentially changing at any time, we need to copy,
		// so we do not change shared structures be accident.
		fieldsNeedsCopy := needsCopy || config.DynamicFields != nil || hasKeyAnyOf(fields, builtin)
		processors.add(makeAddFieldsProcessor("fields", fields, fieldsNeedsCopy))
	}

	if config.DynamicFields != nil {
		checkCopy := func(m common.MapStr) bool {
			return needsCopy || hasKeyAnyOf(m, builtin)
		}
		processors.add(makeAddDynMetaProcessor("dynamicFields", config.DynamicFields, checkCopy))
	}

	// setup 5: client processor list
	processors.add(localProcessors)

	// setup 6: add beats and host metadata
	if len(builtin) > 0 {
		processors.add(makeAddFieldsProcessor("beatsMeta", builtin, needsCopy))
	}

	// setup 7: pipeline processors list
	processors.add(global.processors)

	// setup 9: debug print final event (P)
	if logp.IsDebug("publish") {
		processors.add(debugPrintProcessor(info))
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

func clientEventMeta(meta common.MapStr, needsCopy bool) *processorFn {
	fn := func(event *beat.Event) { addMeta(event, meta) }
	if needsCopy {
		fn = func(event *beat.Event) { addMeta(event, meta.Clone()) }
	}
	return newAnnotateProcessor("@metadata", fn)
}

func addMeta(event *beat.Event, meta common.MapStr) {
	if event.Meta == nil {
		event.Meta = meta
	} else {
		event.Meta.Clone()
		event.Meta.DeepUpdate(meta)
	}
}

func pipelineEventFields(fields common.MapStr, copy bool) *processorFn {
	return makeAddFieldsProcessor("pipelineFields", fields, copy)
}

func makeAddTagsProcessor(name string, tags []string) *processorFn {
	return newAnnotateProcessor(name, func(event *beat.Event) {
		common.AddTags(event.Fields, tags)
	})
}

func makeAddFieldsProcessor(name string, fields common.MapStr, copy bool) *processorFn {
	fn := func(event *beat.Event) { event.Fields.DeepUpdate(fields) }
	if copy {
		fn = func(event *beat.Event) { event.Fields.DeepUpdate(fields.Clone()) }
	}

	return newAnnotateProcessor(name, fn)
}

func makeAddDynMetaProcessor(
	name string,
	meta *common.MapStrPointer,
	checkCopy func(m common.MapStr) bool,
) *processorFn {
	return newAnnotateProcessor(name, func(event *beat.Event) {
		dynFields := meta.Get()
		if checkCopy(dynFields) {
			dynFields = dynFields.Clone()
		}

		event.Fields.DeepUpdate(dynFields)
	})
}

func debugPrintProcessor(info beat.Info) *processorFn {
	// ensure only one go-routine is using the encoder (in case
	// beat.Client is shared between multiple go-routines by accident)
	var mux sync.Mutex

	encoder := json.New(true, false, info.Version)
	return newProcessor("debugPrint", func(event *beat.Event) (*beat.Event, error) {
		mux.Lock()
		defer mux.Unlock()

		b, err := encoder.Encode(info.Beat, event)
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

func hasKey(m common.MapStr, key string) bool {
	_, exists := m[key]
	return exists
}

func hasKeyAnyOf(m, builtin common.MapStr) bool {
	for k := range builtin {
		if hasKey(m, k) {
			return true
		}
	}
	return false
}
