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

package addfields

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type addFields struct {
	fields    mapstr.M
	shared    bool
	overwrite bool

	// metaFields contains only the @metadata value when fields has @metadata
	// but no @timestamp. This allows splitting the update into a fast-path
	// Fields.DeepUpdate + a targeted Meta update, avoiding the overhead of
	// event.deepUpdate's delete/defer pattern.
	metaFields mapstr.M

	// fieldsOnly contains the fields without @metadata/@timestamp keys.
	// Used together with metaFields to avoid the generic deepUpdate path.
	fieldsOnly mapstr.M

	// singleKey is set when the fields map has exactly one top-level key
	// wrapping an inner mapstr.M (e.g. {"elastic_agent": {"id": "...", ...}}).
	// This is the dominant shape created by MakeFieldsProcessor/generateAddFieldsProcessor.
	// When set, Run() clones only the inner map and builds a temporary wrapper,
	// saving one map allocation per event vs cloning the entire tree.
	singleKey      string
	singleKeyInner mapstr.M
}

// FieldsKey is the default target key for the add_fields processor.
const FieldsKey = "fields"

// CreateAddFields constructs an add_fields processor from config.
func CreateAddFields(c *conf.C, _ *logp.Logger) (beat.Processor, error) {
	config := struct {
		Fields mapstr.M `config:"fields" validate:"required"`
		Target *string  `config:"target"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields configuration: %w", err)
	}

	return MakeFieldsProcessor(
		optTarget(config.Target, FieldsKey),
		config.Fields,
		true,
	), nil
}

// NewAddFields creates a new processor adding the given fields to events.
// Set `shared` true if there is the chance of labels being changed/modified by
// subsequent processors.
func NewAddFields(fields mapstr.M, shared bool, overwrite bool) beat.Processor {
	_, hasTimestamp := fields[beat.TimestampFieldKey]
	metaValue, hasMeta := fields[beat.MetadataFieldKey]

	af := &addFields{
		fields:    fields,
		shared:    shared,
		overwrite: overwrite,
	}

	// Pre-split fields with @metadata but no @timestamp for the optimized path.
	if hasMeta && !hasTimestamp {
		if metaMap, ok := metaValue.(mapstr.M); ok {
			af.metaFields = metaMap
			if len(fields) > 1 {
				af.fieldsOnly = make(mapstr.M, len(fields)-1)
				for k, v := range fields {
					if k != beat.MetadataFieldKey {
						af.fieldsOnly[k] = v
					}
				}
			}
		}
	}

	// Detect single-key wrapper shape: {"target": mapstr.M{...}}.
	// This is the dominant pattern from MakeFieldsProcessor and elastic agent.
	// When shared=true, we only need to clone the inner map, not the outer wrapper.
	if shared && !hasTimestamp && !hasMeta && len(fields) == 1 {
		for k, v := range fields {
			if inner, ok := v.(mapstr.M); ok {
				af.singleKey = k
				af.singleKeyInner = inner
			}
		}
	}

	return af
}

func (af *addFields) Run(event *beat.Event) (*beat.Event, error) {
	if event == nil || len(af.fields) == 0 {
		return event, nil
	}

	// Single-key wrapper fast path: when fields have exactly one top-level key
	// wrapping a nested mapstr.M (e.g. {"elastic_agent": {"id": "...", ...}}),
	// clone only the inner map and build a temporary wrapper. This avoids
	// cloning the outer map and bypasses event.deepUpdate's special key checks.
	// This is the dominant shape from MakeFieldsProcessor and elastic agent.
	if af.singleKeyInner != nil {
		if event.Fields == nil {
			event.Fields = mapstr.M{}
		}
		if af.shared && af.overwrite {
			event.Fields.DeepCloneUpdate(mapstr.M{af.singleKey: af.singleKeyInner})
		} else if af.shared {
			event.Fields.DeepCloneUpdateNoOverwrite(mapstr.M{af.singleKey: af.singleKeyInner})
		} else if af.overwrite {
			event.Fields.DeepUpdate(mapstr.M{af.singleKey: af.singleKeyInner})
		} else {
			event.Fields.DeepUpdateNoOverwrite(mapstr.M{af.singleKey: af.singleKeyInner})
		}
		return event, nil
	}

	// Metadata split path: when fields contain @metadata but no @timestamp,
	// update event.Meta and event.Fields separately. This avoids cloning the
	// outer {"@metadata": inner} wrapper and bypasses event.deepUpdate's
	// delete/defer pattern for @metadata handling.
	if af.metaFields != nil {
		if event.Meta == nil {
			event.Meta = mapstr.M{}
		}
		if af.shared && af.overwrite {
			event.Meta.DeepCloneUpdate(af.metaFields)
		} else if af.shared {
			event.Meta.DeepCloneUpdateNoOverwrite(af.metaFields)
		} else if af.overwrite {
			event.Meta.DeepUpdate(af.metaFields)
		} else {
			event.Meta.DeepUpdateNoOverwrite(af.metaFields)
		}
		if len(af.fieldsOnly) > 0 {
			if event.Fields == nil {
				event.Fields = mapstr.M{}
			}
			if af.shared && af.overwrite {
				event.Fields.DeepCloneUpdate(af.fieldsOnly)
			} else if af.shared {
				event.Fields.DeepCloneUpdateNoOverwrite(af.fieldsOnly)
			} else if af.overwrite {
				event.Fields.DeepUpdate(af.fieldsOnly)
			} else {
				event.Fields.DeepUpdateNoOverwrite(af.fieldsOnly)
			}
		}
		return event, nil
	}

	// General path: handles @timestamp, @metadata, and regular fields.
	if event.Fields == nil {
		event.Fields = mapstr.M{}
	}

	_, hasTimestamp := af.fields[beat.TimestampFieldKey]
	_, hasMeta := af.fields[beat.MetadataFieldKey]

	if !hasTimestamp && !hasMeta && af.shared {
		// No special keys — safe to merge directly into Fields.
		if af.overwrite {
			event.Fields.DeepCloneUpdate(af.fields)
		} else {
			event.Fields.DeepCloneUpdateNoOverwrite(af.fields)
		}
		return event, nil
	}

	// Slow path: has @timestamp or @metadata, needs event.DeepUpdate
	// which handles those special keys.
	fields := af.fields
	if af.shared {
		fields = fields.Clone()
	}
	if af.overwrite {
		event.DeepUpdate(fields)
	} else {
		event.DeepUpdateNoOverwrite(fields)
	}
	return event, nil
}

func (af *addFields) String() string {
	s, _ := json.Marshal(af.fields)
	return fmt.Sprintf("add_fields=%s", s)
}

func optTarget(opt *string, def string) string {
	if opt == nil {
		return def
	}
	return *opt
}

func MakeFieldsProcessor(target string, fields mapstr.M, shared bool) beat.Processor {
	if target != "" {
		fields = mapstr.M{
			target: fields,
		}
	}

	return NewAddFields(fields, shared, true)
}
