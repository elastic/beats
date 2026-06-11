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
	// Consume initOnce so that getMeta → init() is a no-op and won't try to
	// fetch metadata from the network or close initDone a second time.
	p.initOnce.Do(func() {})
	return p
}

// TestRunPdataOverwriteFalse verifies that when overwrite=false an existing
// field in the pcommon.Map is not replaced by cloud metadata.
func TestRunPdataOverwriteFalse(t *testing.T) {
	meta := mapstr.M{
		"cloud.provider":    "aws",
		"cloud.instance.id": "i-12345",
	}
	p := newPreloadedProcessor(t, meta, false)

	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, mapstr.M{
		"cloud": mapstr.M{"provider": "existing-provider"},
	}))

	require.NoError(t, p.RunPdata(body))

	out := otelmap.ToMapstr(body)
	cloud, _ := out["cloud"].(map[string]interface{})
	assert.Equal(t, "existing-provider", cloud["provider"], "overwrite=false must preserve existing field")
	assert.Equal(t, "i-12345", cloud["instance"].(map[string]interface{})["id"], "overwrite=false must add absent field")
}

// TestRunPdataOverwriteTrue verifies that when overwrite=true an existing field
// in the pcommon.Map is replaced by cloud metadata.
func TestRunPdataOverwriteTrue(t *testing.T) {
	meta := mapstr.M{
		"cloud.provider": "aws",
	}
	p := newPreloadedProcessor(t, meta, true)

	body := pcommon.NewMap()
	require.NoError(t, otelmap.FromMapstr(body, mapstr.M{
		"cloud": mapstr.M{"provider": "existing-provider"},
	}))

	require.NoError(t, p.RunPdata(body))

	out := otelmap.ToMapstr(body)
	cloud, _ := out["cloud"].(map[string]interface{})
	assert.Equal(t, "aws", cloud["provider"], "overwrite=true must replace existing field")
}
