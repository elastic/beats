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

package processing

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions"
)

type pipelineProcessors struct {
	info beat.Info
	log  *logp.Logger

	skipNormalize bool

	// The pipeline its processor settings for
	// constructing the clients complete processor
	// pipeline on connect.
	modifiers   []modifier
	builtinMeta common.MapStr
	fields      common.MapStr
	tags        []string

	processors beat.Processor

	drop       bool // disabled is set if outputs have been disabled via CLI
	alwaysCopy bool
}

const ecsVersion = "1.0.0-beta2"

type modifier interface {
	// BuiltinFields defines global fields to be added to every event.
	BuiltinFields(beat.Info) common.MapStr

	// ClientFields defines connection local fields to be added to each event
	// of a pipieline client.
	ClientFields(beat.Info, beat.ProcessingConfig) common.MapStr
}

type builtinModifier func(beat.Info) common.MapStr

// NewBeatSupport creates a new SupporterFactory based on NewDefaultSupport.
// NewBeatSupport automatically adds the `ecs.version`, `host.name` and `agent.X` fields
// to each event.
func NewBeatSupport() SupporterFactory {
	return NewDefaultSupport(true, WithECS, WithHost, WithBeatMeta("agent"))
}

// NewObserverSupport creates a new SupporterFactory based on NewDefaultSupport.
// NewObserverSupport automatically adds the `ecs.version` and `observer.X` fields
// to each event.
func NewObserverSupport(normalize bool) SupporterFactory {
	return NewDefaultSupport(normalize, WithECS, WithBeatMeta("observer"))
}

// NewDefaultSupport creates a new SupporterFactory for use with the publisher pipeline.
// If normalize is set, events will be normalized first before being presented
// to the actual processors.
// The Supporter will apply the global `fields`, `fields_under_root`, `tags`
// and `processor` settings to the event processing pipeline to be generated.
// Use WithFields, WithBeatMeta, and other to declare the builtin fields to be added
// to each event. Builtin fields can be modified using global `processors`, and `fields` only.
func NewDefaultSupport(
	normalize bool,
	modifiers ...modifier,
) SupporterFactory {
	return func(info beat.Info, log *logp.Logger, beatCfg *common.Config) (Supporter, error) {
		cfg := struct {
			common.EventMetadata `config:",inline"`      // Fields and tags to add to each event.
			Processors           processors.PluginConfig `config:"processors"`
		}{}
		if err := beatCfg.Unpack(&cfg); err != nil {
			return nil, err
		}

		processors, err := processors.New(cfg.Processors)
		if err != nil {
			return nil, fmt.Errorf("error initializing processors: %v", err)
		}

		p := pipelineProcessors{
			skipNormalize: !normalize,
			log:           log,
		}

		hasProcessors := processors != nil && len(processors.List) > 0
		if hasProcessors {
			tmp := newProgram("global", log)
			for _, p := range processors.List {
				tmp.add(p)
			}
			p.processors = tmp
		}

		builtin := common.MapStr{}
		for _, mod := range modifiers {
			m := mod.BuiltinFields(info)
			if len(m) > 0 {
				builtin.DeepUpdate(m.Clone())
			}
		}
		if len(builtin) > 0 {
			p.builtinMeta = builtin
		}

		if em := cfg.EventMetadata; len(em.Fields) > 0 {
			fields := common.MapStr{}
			common.MergeFields(fields, em.Fields.Clone(), em.FieldsUnderRoot)
			p.fields = fields
		}

		if t := cfg.EventMetadata.Tags; len(t) > 0 {
			p.tags = t
		}

		return p.build, nil
	}
}

// WithFields creates a modifier with the given default builtin fields.
func WithFields(fields common.MapStr) modifier {
	return builtinModifier(func(_ beat.Info) common.MapStr {
		return fields
	})
}

// WithECS modifier adds `ecs.version` builtin fields to a processing pipeline.
var WithECS modifier = WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecsVersion,
	},
})

// WithHost modifier adds `host.name` builtin fields to a processing pipeline
var WithHost modifier = builtinModifier(func(info beat.Info) common.MapStr {
	return common.MapStr{
		"host": common.MapStr{
			"name": info.Name,
		},
	}
})

// WithBeatMeta adds beat meta information as builtin fields to a processing pipeline.
// The `key` parameter defines the field to be used.
func WithBeatMeta(key string) modifier {
	return builtinModifier(func(info beat.Info) common.MapStr {
		metadata := common.MapStr{
			"type":         info.Beat,
			"ephemeral_id": info.EphemeralID.String(),
			"hostname":     info.Hostname,
			"id":           info.ID.String(),
			"version":      info.Version,
		}
		if info.Name != info.Hostname {
			metadata.Put("name", info.Name)
		}
		return common.MapStr{key: metadata}
	})
}

// build prepares the processor pipeline, merging
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
//  7. (P) add builtins
//  8. (P) pipeline processors list
//  9. (P) (if publish/debug enabled) log event
// 10. (P) (if output disabled) dropEvent
func (pp *pipelineProcessors) build(
	cfg beat.ProcessingConfig,
	drop bool,
) (beat.Processor, error) {
	var (
		// pipeline processors
		processors = &program{
			title: "processPipeline",
			log:   pp.log,
		}

		// client fields and metadata
		clientMeta      = cfg.Meta
		localProcessors = makeClientProcessors(pp.log, cfg)
	)

	needsCopy := pp.alwaysCopy || localProcessors != nil || pp.processors != nil

	builtin := pp.builtinMeta
	var clientFields common.MapStr
	for _, mod := range pp.modifiers {
		m := mod.ClientFields(pp.info, cfg)
		if len(m) > 0 {
			if clientFields == nil {
				clientFields = common.MapStr{}
			}
			clientFields.DeepUpdate(m.Clone())
		}
	}
	if len(clientFields) > 0 {
		tmp := builtin.Clone()
		tmp.DeepUpdate(clientFields)
		builtin = tmp
	}

	if !pp.skipNormalize {
		// setup 1: generalize/normalize output (P)
		processors.add(generalizeProcessor)
	}

	// setup 2: add Meta from client config (C)
	if m := clientMeta; len(m) > 0 {
		processors.add(clientEventMeta(m, needsCopy))
	}

	// setup 4, 5: pipeline tags + client tags
	var tags []string
	tags = append(tags, pp.tags...)
	tags = append(tags, cfg.EventMetadata.Tags...)
	if len(tags) > 0 {
		processors.add(actions.NewAddTags("tags", tags))
	}

	// setup 3, 4, 5: client config fields + pipeline fields + client fields + dyn metadata
	fields := cfg.Fields.Clone()
	fields.DeepUpdate(pp.fields.Clone())
	if em := cfg.EventMetadata; len(em.Fields) > 0 {
		common.MergeFields(fields, em.Fields.Clone(), em.FieldsUnderRoot)
	}

	if len(fields) > 0 {
		// Enforce a copy of fields if dynamic fields are configured or agent
		// metadata will be merged into the fields.
		// With dynamic fields potentially changing at any time, we need to copy,
		// so we do not change shared structures be accident.
		fieldsNeedsCopy := needsCopy || cfg.DynamicFields != nil || hasKeyAnyOf(fields, builtin)
		processors.add(actions.NewAddFields(fields, fieldsNeedsCopy))
	}

	if cfg.DynamicFields != nil {
		checkCopy := func(m common.MapStr) bool {
			return needsCopy || hasKeyAnyOf(m, builtin)
		}
		processors.add(makeAddDynMetaProcessor("dynamicFields", cfg.DynamicFields, checkCopy))
	}

	// setup 5: client processor list
	processors.add(localProcessors)

	// setup 6: add beats and host metadata
	if meta := builtin; len(meta) > 0 {
		processors.add(actions.NewAddFields(meta, needsCopy))
	}

	// setup 8: pipeline processors list
	processors.add(pp.processors)

	// setup 9: debug print final event (P)
	if pp.log.IsDebug() {
		processors.add(debugPrintProcessor(pp.info, pp.log))
	}

	// setup 10: drop all events if outputs are disabled (P)
	if drop {
		processors.add(dropDisabledProcessor)
	}

	return processors, nil
}

func makeClientProcessors(
	log *logp.Logger,
	cfg beat.ProcessingConfig,
) processors.Processor {
	procs := cfg.Processor
	if procs == nil || len(procs.All()) == 0 {
		return nil
	}

	p := newProgram("client", log)
	p.list = procs.All()
	return p
}

func (b builtinModifier) BuiltinFields(info beat.Info) common.MapStr {
	return b(info)
}

func (b builtinModifier) ClientFields(_ beat.Info, _ beat.ProcessingConfig) common.MapStr {
	return nil
}
