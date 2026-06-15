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

//go:build (linux || darwin || windows) && !integration

package add_docker_metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestRunPdataParityMatchFields verifies that RunPdata and Run produce the same
// container metadata when the CID is resolved via user-defined match_fields.
func TestRunPdataParityMatchFields(t *testing.T) {
	containers := map[string]*docker.Container{
		"abc123": {
			ID:    "abc123",
			Image: "myimage:latest",
			Name:  "mycontainer",
			Labels: map[string]string{
				"com.example.key": "value",
			},
		},
	}
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"container.id"},
		"labels.dedot": false,
	})
	require.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), cfg, MockWatcherFactory(containers, nil))
	require.NoError(t, err)

	input := mapstr.M{"container": mapstr.M{"id": "abc123"}}

	// Legacy Run path.
	legacyEvent, err := p.Run(&beat.Event{Fields: input.Clone()})
	require.NoError(t, err)
	legacyFields := roundTripNorm(t, legacyEvent.Fields)

	// RunPdata path.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input.Clone()))
	proc := p.(*addDockerMetadata)
	require.NoError(t, proc.RunPdata(body))
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"Run and RunPdata must produce identical output fields")
}

// TestRunPdataNoMatch verifies that RunPdata leaves the body unchanged when the
// container ID is not in the watcher's registry.
func TestRunPdataNoMatch(t *testing.T) {
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"container.id"},
	})
	require.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), cfg, MockWatcherFactory(nil, nil))
	require.NoError(t, err)

	input := mapstr.M{"container": mapstr.M{"id": "notfound"}}
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, input.Clone()))

	proc := p.(*addDockerMetadata)
	require.NoError(t, proc.RunPdata(body))

	assert.Equal(t, roundTripNorm(t, input), otelmap.ToMapstr(body), "body must be unchanged on no match")
}

// roundTripNorm normalises a mapstr.M through a pdata round-trip so that nested
// maps use map[string]interface{}, matching the output of otelmap.ToMapstr.
func roundTripNorm(t *testing.T, m mapstr.M) mapstr.M {
	t.Helper()
	tmp := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(tmp, m))
	return otelmap.ToMapstr(tmp)
}
