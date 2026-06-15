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

package add_host_metadata

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestAddHostMetadataRunParityBasic verifies that the legacy Run path and the
// pdata RunPdata path produce identical host.* fields for a plain empty event.
func TestAddHostMetadataRunParityBasic(t *testing.T) {
	switch runtime.GOOS {
	case "windows", "darwin", "linux", "solaris":
		// supported
	default:
		t.Skip("add_host_metadata not implemented on this OS")
	}

	factory := func() (hostInfo, error) {
		return &mockHostInfo{Hostname: "test-host"}, nil
	}
	testConfig, err := conf.NewConfigFrom(map[string]interface{}{
		"netinfo.enabled": false,
	})
	require.NoError(t, err)

	p, err := newWithHostInfoFactory(testConfig, logptest.NewTestingLogger(t, ""), factory)
	require.NoError(t, err)

	// Legacy Run path.
	legacyEvent := &beat.Event{Fields: mapstr.M{}}
	legacyOut, err := p.Run(legacyEvent)
	require.NoError(t, err)
	// Normalize legacy fields through pdata to get consistent map types.
	legacyNorm := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(legacyNorm, legacyOut.Fields))
	legacyFields := otelmap.ToMapstr(legacyNorm)

	// RunPdata path.
	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, mapstr.M{}))
	pdataProc, ok := p.(interface{ RunPdata(pcommon.Map) error })
	require.True(t, ok, "processor must implement RunPdata")
	require.NoError(t, pdataProc.RunPdata(body))
	pdataFields := otelmap.ToMapstr(body)

	assert.Equal(t, legacyFields, pdataFields,
		"legacy Run and RunPdata must produce identical output fields")
}
