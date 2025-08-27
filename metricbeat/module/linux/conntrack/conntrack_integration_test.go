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

//go:build integration && linux

package conntrack

import (
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/linux"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func BenchmarkFetchNetlink(b *testing.B) {
	if os.Getuid() != 0 {
		b.Skip("This benchmark requires CAP_NET_ADMIN capability (run as root)")
		return
	}

	cfg := getConfig()
	cfg["hostfs"] = b.TempDir()
	f := mbtest.NewReportingMetricSetV2Error(b, cfg)
	for range b.N {
		_, errs := mbtest.ReportingFetchV2Error(f)
		require.Empty(b, errs, "fetch should not return an error")
	}
}

func TestFetchNetlink(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("This test requires CAP_NET_ADMIN capability (run as root)")
		return
	}

	// hide /proc/net/stat/nf_conntrack file so it uses netlink
	cfg := getConfig()
	cfg["hostfs"] = t.TempDir()

	f := mbtest.NewReportingMetricSetV2Error(t, cfg)
	events, errs := mbtest.ReportingFetchV2Error(f)
	require.Empty(t, errs, "fetch should not return an error")

	require.NotEmpty(t, events)
	rawEvent := events[0].BeatEvent("linux", "conntrack").Fields["linux"].(mapstr.M)["conntrack"].(mapstr.M)["summary"].(mapstr.M) //nolint:errcheck // ignore
	keys := slices.Collect(maps.Keys(rawEvent))
	assert.Contains(t, keys, "entries")
	assert.Greater(t, rawEvent["entries"], uint64(0), "entries should be greater than 0")
	assert.Contains(t, keys, "drop")
	assert.Contains(t, keys, "early_drop")
	assert.Contains(t, keys, "found")
	assert.Contains(t, keys, "ignore")
	assert.Contains(t, keys, "insert_failed")
	assert.Contains(t, keys, "invalid")
	assert.Contains(t, keys, "search_restart")
}

func TestFetchProcfsVsNetlink(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("This test requires CAP_NET_ADMIN capability (run as root)")
		return
	}

	cfg := getConfig()
	cfg["hostfs"] = "/"

	// procfs
	f := mbtest.NewReportingMetricSetV2Error(t, cfg)
	events, errs := mbtest.ReportingFetchV2Error(f)
	require.Empty(t, errs, "fetch should not return an error")
	require.NotEmpty(t, events)
	procfsEvent := events[0].BeatEvent("linux", "conntrack").Fields["linux"].(mapstr.M)["conntrack"].(mapstr.M)["summary"].(mapstr.M) //nolint:errcheck // ignore

	// netlink
	cfg["hostfs"] = t.TempDir()

	f = mbtest.NewReportingMetricSetV2Error(t, cfg)
	events, errs = mbtest.ReportingFetchV2Error(f)
	require.Empty(t, errs, "fetch should not return an error")

	require.NotEmpty(t, events)
	netlinkEvent := events[0].BeatEvent("linux", "conntrack").Fields["linux"].(mapstr.M)["conntrack"].(mapstr.M)["summary"].(mapstr.M) //nolint:errcheck // ignore

	assert.Equal(t, procfsEvent, netlinkEvent, "events should be equal")
}
