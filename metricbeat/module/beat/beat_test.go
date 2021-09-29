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

<<<<<<< HEAD
package beat_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/beat"

	// Make sure metricsets are registered in mb.Registry
	_ "github.com/elastic/beats/v7/metricbeat/module/beat/state"
	_ "github.com/elastic/beats/v7/metricbeat/module/beat/stats"
)

func TestXPackEnabledMetricsets(t *testing.T) {
	config := map[string]interface{}{
		"module":        beat.ModuleName,
		"hosts":         []string{"foobar:5066"},
		"xpack.enabled": true,
	}

	metricSets := mbtest.NewReportingMetricSetV2Errors(t, config)
	require.Len(t, metricSets, 2)
	for _, ms := range metricSets {
		name := ms.Name()
		switch name {
		case "state", "stats":
		default:
			t.Errorf("unexpected metricset name = %v", name)
		}
=======
package beat

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestFetchURI(t *testing.T) {
	tcs := []struct {
		orig, path, want string
	}{
		{
			orig: "https://localhost:5000/some/proxy/path",
			path: "/state",
			want: "https://localhost:5000/some/proxy/path/state",
		}, {
			orig: "https://localhost:5000/some/proxy/path/state",
			path: "/state",
			want: "https://localhost:5000/some/proxy/path/state",
		}, {
			orig: "https://localhost:5000/some/proxy/path/state",
			path: "/",
			want: "https://localhost:5000/some/proxy/path",
		}, {
			orig: "http://localhost:5000",
			path: "/state",
			want: "http://localhost:5000/state",
		}, {
			orig: "http://localhost:5000/state",
			path: "/state",
			want: "http://localhost:5000/state",
		}, {
			orig: "http://localhost:5000/stats",
			path: "/state",
			want: "http://localhost:5000/state",
		}, {
			orig: "http://localhost:5000/stats",
			path: "/",
			want: "http://localhost:5000/",
		}, {
			orig: "http://localhost:5000/state",
			path: "/",
			want: "http://localhost:5000/",
		},
	}

	for _, tc := range tcs {
		u, err := url.Parse(tc.orig)
		require.NoError(t, err)
		got := fetchURI(u, tc.path)
		assert.Equal(t, tc.want, got)
>>>>>>> dda584849f ([metricbeat] support basepath in beat module (#28162))
	}
}
