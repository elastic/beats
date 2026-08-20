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

package add_cloud_metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// newPreloadedProcessor builds an addCloudMetadata with metadata already set,
// bypassing the network fetch. initDone is closed so getMeta returns immediately.
func newPreloadedProcessor(t *testing.T, meta mapstr.M, overwrite bool) *addCloudMetadata {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	p := &addCloudMetadata{
		baseCtx:       ctx,
		baseCtxCancel: cancel,
		initData:      &initData{overwrite: overwrite},
		initDone:      make(chan struct{}),
		metadata:      meta,
		logger:        logptest.NewTestingLogger(t, ""),
	}
	// Consume initOnce so getMeta → init() is a no-op: init() never runs, so
	// it never closes initDone or fetches from the network.
	p.initOnce.Do(func() {})
	return p
}

// TestRunPdata verifies that Run and RunPdata produce identical output for the
// overwrite=false and overwrite=true cases, using a preloaded processor that
// bypasses the network fetch.
func TestRunPdata(t *testing.T) {
	cases := []struct {
		name      string
		meta      mapstr.M
		input     mapstr.M
		overwrite bool
	}{
		{
			name: "overwrite=false preserves existing field and adds absent field",
			meta: mapstr.M{
				"cloud.provider":    "aws",
				"cloud.instance.id": "i-12345",
			},
			input:     mapstr.M{"cloud": mapstr.M{"provider": "existing-provider"}},
			overwrite: false,
		},
		{
			name:      "overwrite=true replaces existing field",
			meta:      mapstr.M{"cloud.provider": "aws"},
			input:     mapstr.M{"cloud": mapstr.M{"provider": "existing-provider"}},
			overwrite: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPreloadedProcessor(t, tc.meta, tc.overwrite)

			// Legacy Run path.
			out, err := p.Run(&beat.Event{Fields: tc.input.Clone()})
			require.NoError(t, err)
			require.NotNil(t, out)

			// RunPdata path.
			body := pcommon.NewMap()
			require.NoError(t, otelmap.FromMapstr(body, tc.input))
			drop, err := p.RunPdata(body)
			require.NoError(t, err)
			require.False(t, drop)

			// Normalize Run output through pdata so nested map types match.
			legacyNorm := pcommon.NewMap()
			require.NoError(t, otelmap.FromMapstr(legacyNorm, out.Fields))
			assert.Equal(t, otelmap.ToMapstr(legacyNorm), otelmap.ToMapstr(body),
				"Run and RunPdata must produce identical output")
		})
	}
}
