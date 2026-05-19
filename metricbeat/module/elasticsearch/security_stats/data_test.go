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
	"encoding/json"
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

	// Every counter the API returns must be present in the emitted event;
	// distinct values 1-7 in the input let us catch any field that gets
	// dropped, renamed, or silently swapped with another.
	for path, want := range map[string]int64{
		"dls.cache.entries.count":   1,
		"dls.cache.memory.bytes":    2,
		"dls.cache.hits.count":      3,
		"dls.cache.misses.count":    4,
		"dls.cache.evictions.count": 5,
		"dls.cache.hits.time.ms":    6,
		"dls.cache.misses.time.ms":  7,
	} {
		got, err := events[0].MetricSetFields.GetValue(path)
		require.NoError(t, err, "metric field %s must be present", path)
		require.EqualValues(t, want, got, "metric field %s mismatch", path)
	}
}

func TestMetricSetFieldsJSONShape(t *testing.T) {
	// GetValue passes regardless of whether MetricSetFields uses dotted literal
	// keys ({"entries.count":1}) or proper nesting ({"entries":{"count":1}}),
	// because mapFind tries the full remaining key as a literal before splitting.
	// This test serialises MetricSetFields to JSON — the document actually sent
	// to Elasticsearch — and asserts the properly nested shape. It will fail if
	// dotted literal keys are used in the mapstr.M literal.
	const nodeID = "shape-node"
	input := []byte(`{"nodes":{"` + nodeID + `":{"roles":{"dls":{"bit_set_cache":{"count":1,"memory_in_bytes":2,"hits":3,"misses":4,"evictions":5,"hits_time_in_millis":6,"misses_time_in_millis":7}}}}}}`)

	reporter := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(reporter, info, input, false, nil))
	events := reporter.GetEvents()
	require.Equal(t, 1, len(events))

	raw, err := json.Marshal(events[0].MetricSetFields)
	require.NoError(t, err)

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &doc))

	want := map[string]interface{}{
		"dls": map[string]interface{}{
			"cache": map[string]interface{}{
				"entries":   map[string]interface{}{"count": float64(1)},
				"memory":    map[string]interface{}{"bytes": float64(2)},
				"hits":      map[string]interface{}{"count": float64(3), "time": map[string]interface{}{"ms": float64(6)}},
				"misses":    map[string]interface{}{"count": float64(4), "time": map[string]interface{}{"ms": float64(7)}},
				"evictions": map[string]interface{}{"count": float64(5)},
			},
		},
	}
	require.Equal(t, want, doc, "MetricSetFields must serialise to properly nested objects, not dotted literal keys")
}

func TestEventShapeWithEnrichment(t *testing.T) {
	// Asserts the full event shape on the path where /_nodes enrichment
	// resolves the node: every identifying module field and every DLS cache
	// counter must land at its expected path. Without this, dropping
	// MetricSetFields entirely (or any individual counter) goes unnoticed.
	const nodeID = "1sFM8cmSROZYhPxVsiWew"
	input := []byte(`{"nodes":{"` + nodeID + `":{"roles":{"dls":{"bit_set_cache":{"count":12,"memory_in_bytes":4096,"hits":8421,"misses":137,"evictions":4,"hits_time_in_millis":51,"misses_time_in_millis":219}}}}}}`)

	reporter := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(reporter, info, input, false, fixtureEnrichment))
	events := reporter.GetEvents()
	require.Equal(t, 1, len(events))
	event := events[0]

	for path, want := range map[string]interface{}{
		"cluster.id":   info.ClusterID,
		"cluster.name": info.ClusterName,
		"node.id":      nodeID,
		"node.name":    "instance-0000000019",
		"node.version": "9.2.0",
	} {
		got, err := event.ModuleFields.GetValue(path)
		require.NoError(t, err, "module field %s must be present", path)
		require.Equal(t, want, got, "module field %s mismatch", path)
	}

	roles, err := event.ModuleFields.GetValue("node.roles")
	require.NoError(t, err, "node.roles must be present")
	require.Equal(t, []string{"data_hot", "ingest"}, roles)

	for path, want := range map[string]int64{
		"dls.cache.entries.count":   12,
		"dls.cache.memory.bytes":    4096,
		"dls.cache.hits.count":      8421,
		"dls.cache.misses.count":    137,
		"dls.cache.evictions.count": 4,
		"dls.cache.hits.time.ms":    51,
		"dls.cache.misses.time.ms":  219,
	} {
		got, err := event.MetricSetFields.GetValue(path)
		require.NoError(t, err, "metric field %s must be present", path)
		require.EqualValues(t, want, got, "metric field %s mismatch", path)
	}
}
