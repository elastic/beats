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

//go:build linux || darwin || windows

package add_kubernetes_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// normMapstr normalizes a mapstr.M through a pdata round-trip so that all
// nested maps use map[string]interface{} (matching the output of ToMapstr).
func normMapstr(t *testing.T, m mapstr.M) mapstr.M {
	t.Helper()
	tmp := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(tmp, m))
	return otelmap.ToMapstr(tmp)
}

// TestAnnotatorRunPdataParity verifies that Run and RunPdata produce the same
// output for a cache hit with full container metadata.
func TestAnnotatorRunPdataParity(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myimage:latest",
				"id":      "abc123",
				"runtime": "containerd",
			},
		},
	}
	processor := newAnnotatorForTest(t, "abc123", meta)

	input := mapstr.M{"container": mapstr.M{"id": "abc123"}}

	// Legacy Run path.
	legacyEvent, err := processor.Run(baseEvent("abc123"))
	require.NoError(t, err)
	legacyFields := normMapstr(t, legacyEvent.Fields)

	// RunPdata path.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	require.NoError(t, processor.RunPdata(body))
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"Run and RunPdata must produce identical output fields")
}

// TestAnnotatorRunPdataParityNoMatch verifies that when no cache entry is found
// both Run and RunPdata leave the event unchanged.
func TestAnnotatorRunPdataParityNoMatch(t *testing.T) {
	meta := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{"name": "mypod"},
		},
	}
	processor := newAnnotatorForTest(t, "other-id", meta)

	input := mapstr.M{"container": mapstr.M{"id": "missing-id"}}

	// Legacy Run path.
	legacyEvent, err := processor.Run(baseEvent("missing-id"))
	require.NoError(t, err)
	legacyFields := normMapstr(t, legacyEvent.Fields)

	// RunPdata path.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input))
	require.NoError(t, processor.RunPdata(body))
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"both paths must leave the event unchanged when there is no cache match")
}
