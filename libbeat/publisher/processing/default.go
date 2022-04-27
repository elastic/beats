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

	"github.com/elastic/beats/v7/libbeat/asset"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/mapping"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/processors/timeseries"
	"github.com/elastic/elastic-agent-libs/config"
)

// builder is used to create the event processing pipeline in Beats.  The
// builder orders and merges global and local (per client) event annotation
// settings, with the configured event processors into one common event
// processor for use with the publisher pipeline.
// Also See: (*builder).Create
type builder struct {
	info beat.Info
	log  *logp.Logger

	skipNormalize bool

	// global pipeline fields and tags configurations
	modifiers   []modifier
	builtinMeta common.MapStr
	fields      common.MapStr
	tags        []string

	// Time series id will be calculated for Events with the TimeSeries flag if this
	// is enabled (disabled by default)
	timeSeries       bool
	timeseriesFields mapping.Fields

	// global pipeline processors
	processors *group

	drop       bool // disabled is set if outputs have been disabled via CLI
	alwaysCopy bool
}

type modifier interface {
	// BuiltinFields defines global fields to be added to every event.
	BuiltinFields(beat.Info) common.MapStr

	// ClientFields defines connection local fields to be added to each event
	// of a pipeline client.
	ClientFields(beat.Info, beat.ProcessingConfig) common.MapStr
}

type builtinModifier func(beat.Info) common.MapStr

// MakeDefaultBeatSupport creates a new SupportFactory based on NewDefaultSupport.
// MakeDefaultBeatSupport automatically adds the `ecs.version`, `host.name` and `agent.X` fields
// to each event.
func MakeDefaultBeatSupport(normalize bool) SupportFactory {
	return MakeDefaultSupport(normalize, WithECS, WithHost, WithAgentMeta())
}

// MakeDefaultObserverSupport creates a new SupportFactory based on NewDefaultSupport.
// MakeDefaultObserverSupport automatically adds the `ecs.version` and `observer.X` fields
// to each event.
func MakeDefaultObserverSupport(normalize bool) SupportFactory {
	return MakeDefaultSupport(normalize, WithECS, WithObserverMeta())
}

// MakeDefaultSupport creates a new SupportFactory for use with the publisher pipeline.
// If normalize is set, events will be normalized first before being presented
// to the actual processors.
// The Supporter will apply the global `fields`, `fields_under_root`, `tags`
// and `processor` settings to the event processing pipeline to be generated.
// Use WithFields, WithBeatMeta, and other to declare the builtin fields to be added
// to each event. Builtin fields can be modified using global `processors`, and `fields` only.
func MakeDefaultSupport(
	normalize bool,
	modifiers ...modifier,
) SupportFactory {
	return func(info beat.Info, log *logp.Logger, beatCfg *config.C) (Supporter, error) {
		cfg := struct {
			common.EventMetadata `config:",inline"`      // Fields and tags to add to each event.
			Processors           processors.PluginConfig `config:"processors"`
			TimeSeries           bool                    `config:"timeseries.enabled"`
		}{}
		if err := beatCfg.Unpack(&cfg); err != nil {
			return nil, err
		}

		processors, err := processors.New(cfg.Processors)
		if err != nil {
			return nil, fmt.Errorf("error initializing processors: %v", err)
		}

		return newBuilder(info, log, processors, cfg.EventMetadata, modifiers, !normalize, cfg.TimeSeries)
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
		"version": ecs.Version,
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

// WithAgentMeta adds agent meta information as builtin fields to a processing
// pipeline.
func WithAgentMeta() modifier {
	return builtinModifier(func(info beat.Info) common.MapStr {
		metadata := common.MapStr{
			"ephemeral_id": info.EphemeralID.String(),
			"id":           info.ID.String(),
			"name":         info.Hostname,
			"type":         info.Beat,
			"version":      info.Version,
		}
		if info.Name != "" {
			metadata["name"] = info.Name
		}
		return common.MapStr{"agent": metadata}
	})
}

// WithObserverMeta adds beat meta information as builtin fields to a processing
// pipeline.
func WithObserverMeta() modifier {
	return builtinModifier(func(info beat.Info) common.MapStr {
		metadata := common.MapStr{
			"type":         info.Beat,                 // Per ECS this is not a valid type value.
			"ephemeral_id": info.EphemeralID.String(), // Not in ECS.
			"hostname":     info.Hostname,
			"id":           info.ID.String(), // Not in ECS.
			"version":      info.Version,
		}
		if info.Name != info.Hostname {
			metadata.Put("name", info.Name)
		}
		return common.MapStr{"observer": metadata}
	})
}

func newBuilder(
	info beat.Info,
	log *logp.Logger,
	processors *processors.Processors,
	eventMeta common.EventMetadata,
	modifiers []modifier,
	skipNormalize bool,
	timeSeries bool,
) (*builder, error) {
	b := &builder{
		skipNormalize: skipNormalize,
		modifiers:     modifiers,
		log:           log,
		info:          info,
		timeSeries:    timeSeries,
	}

	hasProcessors := processors != nil && len(processors.List) > 0
	if hasProcessors {
		tmp := newGroup("global", log)
		for _, p := range processors.List {
			tmp.add(p)
		}
		b.processors = tmp
	}

	builtin := common.MapStr{}
	for _, mod := range modifiers {
		m := mod.BuiltinFields(info)
		if len(m) > 0 {
			builtin.DeepUpdate(m.Clone())
		}
	}
	if len(builtin) > 0 {
		b.builtinMeta = builtin
	}

	if fields := eventMeta.Fields; len(fields) > 0 {
		b.fields = common.MapStr{}
		common.MergeFields(b.fields, fields.Clone(), eventMeta.FieldsUnderRoot)
	}

	if timeSeries {
		rawFields, err := asset.GetFields(info.Beat)
		if err != nil {
			return nil, err
		}

		fields, err := mapping.LoadFields(rawFields)
		if err != nil {
			return nil, err
		}

		b.timeseriesFields = fields
	}

	if t := eventMeta.Tags; len(t) > 0 {
		b.tags = t
	}

	return b, nil
}

// Create combines the builder configuration with the client settings
// in order to build the event processing pipeline.
//
// Processing order (C=client, P=pipeline)
//  1. (P) generalize/normalize event
//  2. (C) add Meta from client Config to event.Meta
//  3. (C) add Fields from client config to event.Fields
//  4. (P) add pipeline fields + tags
//  5. (C) add client fields + tags
//  6. (C) client processors list
//  7. (P) add builtins
//  8. (P) pipeline processors list
//  9. (P) timeseries mangling
//  10. (P) (if publish/debug enabled) log event
//  11. (P) (if output disabled) dropEvent
func (b *builder) Create(cfg beat.ProcessingConfig, drop bool) (beat.Processor, error) {
	var (
		// pipeline processors
		processors = newGroup("processPipeline", b.log)

		// client fields and metadata
		clientMeta      = cfg.Meta
		localProcessors = makeClientProcessors(b.log, cfg)
	)

	needsCopy := b.alwaysCopy || localProcessors != nil || b.processors != nil

	builtin := b.builtinMeta
	if cfg.DisableHost {
		tmp := builtin.Clone()
		tmp.Delete("host")
		builtin = tmp
	}

	var clientFields common.MapStr
	for _, mod := range b.modifiers {
		m := mod.ClientFields(b.info, cfg)
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

	if !b.skipNormalize {
		// setup 1: generalize/normalize output (P)
		processors.add(newGeneralizeProcessor(cfg.KeepNull))
	}

	// setup 2: add Meta from client config (C)
	if m := clientMeta; len(m) > 0 {
		processors.add(clientEventMeta(m, needsCopy))
	}

	// setup 4, 5: pipeline tags + client tags
	var tags []string
	tags = append(tags, b.tags...)
	tags = append(tags, cfg.EventMetadata.Tags...)
	if len(tags) > 0 {
		processors.add(actions.NewAddTags("tags", tags))
	}

	// setup 3, 4, 5: client config fields + pipeline fields + client fields + dyn metadata
	fields := cfg.Fields.Clone()
	fields.DeepUpdate(b.fields.Clone())
	if em := cfg.EventMetadata; len(em.Fields) > 0 {
		common.MergeFieldsDeep(fields, em.Fields.Clone(), em.FieldsUnderRoot)
	}

	if len(fields) > 0 {
		// Enforce a copy of fields if dynamic fields are configured or agent
		// metadata will be merged into the fields.
		// With dynamic fields potentially changing at any time, we need to copy,
		// so we do not change shared structures be accident.
		fieldsNeedsCopy := needsCopy || cfg.DynamicFields != nil || hasKeyAnyOf(fields, builtin)
		processors.add(actions.NewAddFields(fields, fieldsNeedsCopy, true))
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
		processors.add(actions.NewAddFields(meta, needsCopy, false))
	}

	// setup 8: pipeline processors list
	if b.processors != nil {
		// Add the global pipeline as a function processor, so clients cannot close it
		processors.add(newProcessor(b.processors.title, b.processors.Run))
	}

	// setup 9: time series metadata
	if b.timeSeries {
		processors.add(timeseries.NewTimeSeriesProcessor(b.timeseriesFields))
	}

	// setup 10: debug print final event (P)
	if b.log.IsDebug() {
		processors.add(debugPrintProcessor(b.info, b.log))
	}

	// setup 11: drop all events if outputs are disabled (P)
	if drop {
		processors.add(dropDisabledProcessor)
	}

	return processors, nil
}

func (b *builder) Close() error {
	if b.processors != nil {
		return b.processors.Close()
	}
	return nil
}

func makeClientProcessors(
	log *logp.Logger,
	cfg beat.ProcessingConfig,
) processors.Processor {
	procs := cfg.Processor
	if procs == nil || len(procs.All()) == 0 {
		return nil
	}

	p := newGroup("client", log)
	p.list = procs.All()
	return p
}

func (b builtinModifier) BuiltinFields(info beat.Info) common.MapStr {
	return b(info)
}

func (b builtinModifier) ClientFields(_ beat.Info, _ beat.ProcessingConfig) common.MapStr {
	return nil
}
