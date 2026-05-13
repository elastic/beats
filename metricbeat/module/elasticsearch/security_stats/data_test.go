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

//go:build !integration

package security_stats

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

// fixtureEnrichment matches a node ID used in the security_stats.*.json
// fixture so the per-event enrichment branch is exercised in TestMapper.
var fixtureEnrichment = map[string]elasticsearch.NodeEnrichment{
	"1sFM8cmSROZYhPxVsiWew": {
		Name:    "instance-0000000019",
		Roles:   []string{"data_hot", "ingest"},
		Version: "9.2.0",
	},
}

func TestMapper(t *testing.T) {
	mapper := func(r mb.ReporterV2, i elasticsearch.Info, content []byte, isXpack bool) error {
		return eventsMapping(r, i, content, isXpack, fixtureEnrichment)
	}
	elasticsearch.TestMapperWithInfo(t, "./_meta/test/security_stats.*.json", mapper)
}

func TestEmpty(t *testing.T) {
	input, err := os.ReadFile("./_meta/test/empty.920.json")
	require.NoError(t, err, "must be able to read the empty fixture")

	reporter := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(reporter, info, input, true, nil), "empty response must not produce a parse error")
	require.Equal(t, 0, len(reporter.GetErrors()), "empty response must produce no errors")
	require.Equal(t, 0, len(reporter.GetEvents()), "empty response must produce no events")
}

func TestSkipsNodesWithoutDLSStats(t *testing.T) {
	// A node that returns "roles":{} (no dls block) must not produce an event.
	input := []byte(`{"nodes":{"node-without-dls":{"roles":{}},"node-without-roles":{}}}`)

	reporter := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(reporter, info, input, true, nil), "absent DLS stats are not an error")
	require.Equal(t, 0, len(reporter.GetErrors()))
	require.Equal(t, 0, len(reporter.GetEvents()), "nodes without DLS stats must be skipped")
}

func TestRejectsMalformedResponse(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}
	err := eventsMapping(reporter, info, []byte("not json"), true, nil)
	require.Error(t, err, "malformed JSON must surface a parse error")
}

func TestEnrichmentMissingForNode(t *testing.T) {
	// The /_nodes call may have failed or returned a stale set: emit the cache
	// counters anyway and just skip the name/roles/version attachment.
	input := []byte(`{"nodes":{"unknown-node":{"roles":{"dls":{"bit_set_cache":{"count":1,"memory_in_bytes":2,"hits":3,"misses":4,"evictions":5,"hits_time_in_millis":6,"misses_time_in_millis":7}}}}}}`)

	reporter := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(reporter, info, input, true, fixtureEnrichment))
	events := reporter.GetEvents()
	require.Equal(t, 1, len(events))

	id, _ := events[0].ModuleFields.GetValue("node.id")
	require.Equal(t, "unknown-node", id)
	_, err := events[0].ModuleFields.GetValue("node.name")
	require.Error(t, err, "missing-from-enrichment node must not have node.name set")
}
