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

package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// normFields encodes fields through a pdata round-trip so that nested maps
// become map[string]interface{}, matching the output of otelmap.ToMapstr.
func normFields(t *testing.T, m mapstr.M) mapstr.M {
	t.Helper()
	tmp := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(tmp, m))
	return otelmap.ToMapstr(tmp)
}

// addFieldsInner is a simple beat.Processor that adds a fixed field.
type addFieldsInner struct {
	key   string
	value string
}

func (a *addFieldsInner) Run(event *beat.Event) (*beat.Event, error) {
	event.Fields[a.key] = a.value
	return event, nil
}

func (a *addFieldsInner) String() string { return "addFieldsInner" }

// metaProcessor is a beat.Processor that adds a fixed key/value to event.Meta.
type metaProcessor struct {
	key   string
	value string
}

func (m *metaProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if event.Meta == nil {
		event.Meta = mapstr.M{}
	}
	event.Meta[m.key] = m.value
	return event, nil
}

func (m *metaProcessor) String() string { return "metaProcessor" }

// dropProcessor is a beat.Processor that drops every event by returning nil.
type dropProcessor struct{}

func (d *dropProcessor) Run(_ *beat.Event) (*beat.Event, error) { return nil, nil }
func (d *dropProcessor) String() string                         { return "dropProcessor" }

// makeWhenProcessor builds a WhenProcessor with an equals condition on field
// "i" matching matchValue, wrapping inner.
func makeWhenProcessor(t *testing.T, matchValue int, inner beat.Processor) *WhenProcessor {
	t.Helper()
	raw, err := conf.NewConfigFrom(map[string]interface{}{
		"equals": map[string]interface{}{"i": matchValue},
	})
	require.NoError(t, err)
	var condConfig conditions.Config
	require.NoError(t, raw.Unpack(&condConfig))
	proc, err := NewConditionRule(condConfig, inner, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	wp, ok := proc.(*WhenProcessor)
	require.True(t, ok, "expected *WhenProcessor")
	return wp
}

// TestWhenProcessorPdataParityConditionFalse verifies that when the condition
// does not match, both Run and RunPdata leave the event unchanged.
func TestWhenProcessorPdataParityConditionFalse(t *testing.T) {
	inner := &addFieldsInner{key: "added", value: "yes"}
	wp := makeWhenProcessor(t, 42, inner)

	input := mapstr.M{"i": 10}

	// Legacy Run path.
	legacyEvent, err := wp.Run(&beat.Event{Fields: input.Clone()})
	require.NoError(t, err)
	legacyFields := normFields(t, legacyEvent.Fields)

	// RunPdata path.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	drop, err := wp.RunPdata(body)
	require.NoError(t, err)
	assert.False(t, drop)
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"condition false: both paths must leave the event unchanged")
	_, hasAdded := pdataFields["added"]
	assert.False(t, hasAdded, "field 'added' must not be present when condition is false")
}

// TestWhenProcessorPdataParityConditionTrue verifies that when the condition
// matches, both Run and RunPdata apply the inner processor identically.
func TestWhenProcessorPdataParityConditionTrue(t *testing.T) {
	inner := &addFieldsInner{key: "added", value: "yes"}
	wp := makeWhenProcessor(t, 10, inner)

	input := mapstr.M{"i": 10}

	// Legacy Run path — inner processor only supports Run, so RunPdata will
	// fall through to the round-trip path in WhenProcessor.RunPdata.
	legacyEvent, err := wp.Run(&beat.Event{Fields: input.Clone()})
	require.NoError(t, err)
	legacyFields := normFields(t, legacyEvent.Fields)

	// RunPdata path.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	drop, err := wp.RunPdata(body)
	require.NoError(t, err)
	assert.False(t, drop)
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"condition true: both paths must produce identical output fields")
	assert.Equal(t, "yes", pdataFields["added"],
		"field 'added' must be set to 'yes' when condition is true")
}

// TestWhenProcessorPdataLegacyFallbackMetadata verifies that @metadata
// (serialized into the body by otelconsumer) is correctly round-tripped when
// the inner processor does not implement PdataProcessor. The fallback path must
// extract @metadata into event.Meta before calling Run, and write it back
// afterward so that changes by the inner processor survive.
func TestWhenProcessorPdataLegacyFallbackMetadata(t *testing.T) {
	// metaProcessor does not implement PdataProcessor, forcing the round-trip.
	wp := makeWhenProcessor(t, 10, &metaProcessor{key: "op_type", value: "index"})

	// Seed the body with @metadata as otelconsumer would.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, mapstr.M{"i": 10}))
	require.NoError(t, otelmap.FromMapstr(body.PutEmptyMap("@metadata"), mapstr.M{"_id": "abc123"}))

	drop, err := wp.RunPdata(body)
	require.NoError(t, err)
	assert.False(t, drop)

	meta, ok := otelmap.ToMapstr(body)["@metadata"].(map[string]interface{})
	require.True(t, ok, "@metadata must survive the round-trip as a map")
	assert.Equal(t, "abc123", meta["_id"], "existing @metadata fields must be preserved")
	assert.Equal(t, "index", meta["op_type"], "inner processor must have written op_type to @metadata")
}

// TestWhenProcessorPdataParityDrop verifies that when the inner processor drops
// the event (Run returns nil), both Run and RunPdata agree: Run returns nil and
// RunPdata produces an empty body.
func TestWhenProcessorPdataParityDrop(t *testing.T) {
	wp := makeWhenProcessor(t, 10, &dropProcessor{})

	input := mapstr.M{"i": 10, "msg": "hello"}

	// Legacy Run path: nil return signals a drop.
	legacyOut, err := wp.Run(&beat.Event{Fields: input.Clone()})
	require.NoError(t, err)
	assert.Nil(t, legacyOut, "Run must return nil when inner processor drops the event")

	// RunPdata path: drop=true signals the event should be dropped.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	drop, err := wp.RunPdata(body)
	require.NoError(t, err)
	assert.True(t, drop, "RunPdata must return drop=true when inner processor drops the event")
}
