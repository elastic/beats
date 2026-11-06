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

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestStatsHandlerExtra(t *testing.T) {
	reg := monitoring.NewRegistry()
	monitoring.NewString(reg.GetOrCreateRegistry("beat").GetOrCreateRegistry("info"), "name").Set("mine")
	monitoring.NewUint(reg.GetOrCreateRegistry("libbeat"), "events").Set(7)

	extra := func() mapstr.M {
		return mapstr.M{
			"beat":   mapstr.M{"info": mapstr.M{"name": "shared"}, "memstats": mapstr.M{"memory_alloc": 5}},
			"system": mapstr.M{"cpu": mapstr.M{"cores": 4}},
		}
	}

	get := func(h http.HandlerFunc) map[string]any {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stats", nil))
		var out map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), "response should be JSON")
		return out
	}

	without := get(makeStatsHandler(reg, nil))
	assert.NotContains(t, without, "system", "no extra metrics without WithStatsExtra")

	with := get(makeStatsHandler(reg, extra))
	beatNS, ok := with["beat"].(map[string]any)
	require.True(t, ok, "response should have a beat namespace, got %v", with)
	info, ok := beatNS["info"].(map[string]any)
	require.True(t, ok, "beat should have an info namespace, got %v", beatNS)
	assert.Equal(t, "mine", info["name"], "metrics already in the registry must not be overwritten")
	assert.Contains(t, beatNS, "memstats", "extra metrics are added")
	assert.Contains(t, with, "system", "extra namespaces are added")
	assert.Contains(t, with, "libbeat", "registry metrics are kept")

	assert.NotPanics(t, func() { get(makeStatsHandler(reg, func() mapstr.M { return nil })) }, "extra may return nil")
}
