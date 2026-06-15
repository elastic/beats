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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// runLegacy runs proc via the legacy Run path and returns the output fields
// normalized through a pdata round-trip so that nested map types match those
// returned by runPdata.
func runLegacy(t *testing.T, proc *addFields, input mapstr.M) mapstr.M {
	t.Helper()
	event := &beat.Event{Fields: input.Clone()}
	out, err := proc.Run(event)
	require.NoError(t, err)
	require.NotNil(t, out)
	// Normalize: encode through pdata so nested maps become map[string]interface{}
	// just like ToMapstr returns.
	normalized := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(normalized, out.Fields))
	return otelmap.ToMapstr(normalized)
}

// runPdata runs proc via the RunPdata path, starting with a pcommon.Map
// populated from input, and returns the result as mapstr.M.
func runPdata(t *testing.T, proc *addFields, input mapstr.M) mapstr.M {
	t.Helper()
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	require.NoError(t, proc.RunPdata(body))
	return otelmap.ToMapstr(body)
}

func TestAddFieldsParityOverwriteTrue(t *testing.T) {
	input := mapstr.M{"key": "old"}
	fields := mapstr.M{"key": "new"}
	proc := &addFields{fields: fields, shared: false, overwrite: true}

	legacyOut := runLegacy(t, proc, input)
	pdataOut := runPdata(t, proc, input)

	assert.Equal(t, legacyOut, pdataOut)
	assert.Equal(t, "new", legacyOut["key"])
}

func TestAddFieldsParityOverwriteFalse(t *testing.T) {
	input := mapstr.M{"key": "old"}
	fields := mapstr.M{"key": "new"}
	proc := &addFields{fields: fields, shared: false, overwrite: false}

	legacyOut := runLegacy(t, proc, input)
	pdataOut := runPdata(t, proc, input)

	assert.Equal(t, legacyOut, pdataOut)
	assert.Equal(t, "old", legacyOut["key"], "overwrite=false must preserve existing field")
}

func TestAddFieldsParitySharedClonesFields(t *testing.T) {
	// shared=true means the processor clones its field map before merging.
	// Both paths must produce identical output, and neither must mutate
	// the processor's internal fields map.
	input := mapstr.M{"existing": "value"}
	fields := mapstr.M{"extra": "added"}
	proc := &addFields{fields: fields, shared: true, overwrite: true}

	legacyOut := runLegacy(t, proc, input)
	pdataOut := runPdata(t, proc, input)

	assert.Equal(t, legacyOut, pdataOut)

	// The original processor fields must not have been mutated.
	assert.Equal(t, mapstr.M{"extra": "added"}, proc.fields)
}
