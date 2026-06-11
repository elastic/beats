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
	require.NoError(t, wp.RunPdata(body))
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
	require.NoError(t, wp.RunPdata(body))
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"condition true: both paths must produce identical output fields")
	assert.Equal(t, "yes", pdataFields["added"],
		"field 'added' must be set to 'yes' when condition is true")
}
