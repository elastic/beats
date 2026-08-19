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

// pdataAddFieldsInner implements both Run and RunPdata.
type pdataAddFieldsInner struct {
	key   string
	value string
}

func (a *pdataAddFieldsInner) Run(event *beat.Event) (*beat.Event, error) {
	event.Fields[a.key] = a.value
	return event, nil
}

func (a *pdataAddFieldsInner) RunPdata(body pcommon.Map) (bool, error) {
	body.PutStr(a.key, a.value)
	return false, nil
}

func (a *pdataAddFieldsInner) String() string { return "pdataAddFieldsInner" }

// pdataDropProcessor implements both Run and RunPdata and drops every event.
type pdataDropProcessor struct{}

func (d *pdataDropProcessor) Run(_ *beat.Event) (*beat.Event, error) { return nil, nil }
func (d *pdataDropProcessor) RunPdata(_ pcommon.Map) (bool, error)   { return true, nil }
func (d *pdataDropProcessor) String() string                         { return "pdataDropProcessor" }

// closingPdataProcessor implements Run, RunPdata, and Close.
type closingPdataProcessor struct {
	pdataAddFieldsInner
	closed bool
}

func (c *closingPdataProcessor) Close() error {
	c.closed = true
	return nil
}

func (c *closingPdataProcessor) String() string { return "closingPdataProcessor" }

// makeWhenPdataProcessor builds a WhenPdataProcessor with an equals condition
// on field "i" matching matchValue, wrapping a pdata-capable inner.
func makeWhenPdataProcessor(t *testing.T, matchValue int, inner beat.Processor) *WhenPdataProcessor {
	t.Helper()
	raw, err := conf.NewConfigFrom(map[string]any{
		"equals": map[string]any{"i": matchValue},
	})
	require.NoError(t, err)
	var condConfig conditions.Config
	require.NoError(t, raw.Unpack(&condConfig))
	proc, err := NewConditionRule(condConfig, inner, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	wp, ok := proc.(*WhenPdataProcessor)
	require.True(t, ok, "expected *WhenPdataProcessor")
	return wp
}

// TestWhenProcessorPdataParityConditionFalse verifies that when the condition
// does not match, both Run and RunPdata leave the event unchanged.
func TestWhenProcessorPdataParityConditionFalse(t *testing.T) {
	inner := &pdataAddFieldsInner{key: "added", value: "yes"}
	wp := makeWhenPdataProcessor(t, 42, inner)

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
	inner := &pdataAddFieldsInner{key: "added", value: "yes"}
	wp := makeWhenPdataProcessor(t, 10, inner)

	input := mapstr.M{"i": 10}

	// Legacy Run path.
	legacyEvent, err := wp.Run(&beat.Event{Fields: input.Clone()})
	require.NoError(t, err)
	legacyFields := normFields(t, legacyEvent.Fields)

	// RunPdata path — delegates directly to inner's RunPdata, no round-trip.
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

// TestWhenLegacyInnerNoPdataPath verifies that NewConditionRule returns a
// *WhenProcessor (not *WhenPdataProcessor) when the inner processor does not
// implement PdataProcessor.
func TestWhenLegacyInnerNoPdataPath(t *testing.T) {
	raw, err := conf.NewConfigFrom(map[string]any{
		"equals": map[string]any{"i": 10},
	})
	require.NoError(t, err)
	var condConfig conditions.Config
	require.NoError(t, raw.Unpack(&condConfig))

	// addFieldsInner is legacy-only (no RunPdata).
	proc, err := NewConditionRule(condConfig, &addFieldsInner{key: "x", value: "y"}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	_, isWhen := proc.(*WhenProcessor)
	require.True(t, isWhen, "legacy inner must produce *WhenProcessor")

	_, isPdata := proc.(PdataProcessor)
	assert.False(t, isPdata, "*WhenProcessor must not implement PdataProcessor")
}

// TestClosingWhenPdataProcessorConstruction verifies that NewConditionRule
// returns a *ClosingWhenPdataProcessor when the inner implements both
// PdataProcessor and Closer, and that Close is forwarded to the inner.
func TestClosingWhenPdataProcessorConstruction(t *testing.T) {
	raw, err := conf.NewConfigFrom(map[string]any{
		"equals": map[string]any{"i": 10},
	})
	require.NoError(t, err)
	var condConfig conditions.Config
	require.NoError(t, raw.Unpack(&condConfig))

	inner := &closingPdataProcessor{pdataAddFieldsInner: pdataAddFieldsInner{key: "added", value: "yes"}}
	proc, err := NewConditionRule(condConfig, inner, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	_, ok := proc.(*ClosingWhenPdataProcessor)
	require.True(t, ok, "expected *ClosingWhenPdataProcessor when inner implements both PdataProcessor and Closer")

	_, isPdata := proc.(PdataProcessor)
	assert.True(t, isPdata, "*ClosingWhenPdataProcessor must implement PdataProcessor")

	_, isCloser := proc.(Closer)
	assert.True(t, isCloser, "*ClosingWhenPdataProcessor must implement Closer")

	require.NoError(t, Close(proc))
	assert.True(t, inner.closed, "Close must be forwarded to the inner processor")
}

// TestWhenProcessorPdataParityDrop verifies that when the inner processor
// drops the event, both Run and RunPdata agree: Run returns nil and RunPdata
// returns drop=true.
func TestWhenProcessorPdataParityDrop(t *testing.T) {
	wp := makeWhenPdataProcessor(t, 10, &pdataDropProcessor{})

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
