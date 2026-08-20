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

package security_stats

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// response models the /_security/stats payload. The "roles" envelope groups
// counters that the API reports per-roles-subsystem and may be absent on nodes
// where that subsystem is unavailable; we flatten the relevant counters out of
// it to keep the published event focused on what consumers actually query.
type response struct {
	Nodes map[string]nodeResponse `json:"nodes"`
}

type nodeResponse struct {
	Roles *rolesStats `json:"roles,omitempty"`
}

type rolesStats struct {
	DLS *dlsStats `json:"dls,omitempty"`
}

// dlsStats unmarshals the API field "bit_set_cache" into our internal "Cache"
// label so the published field path stays a generic "dls.cache.*"
type dlsStats struct {
	Cache *cacheStats `json:"bit_set_cache,omitempty"`
}

// cacheStats mirrors the per-node DLS cache counters returned by
// /_security/stats. The "memory" string field returned by the API is
// intentionally omitted in favor of "memory_in_bytes" so the emitted event is
// fully numeric and aggregatable.
type cacheStats struct {
	Count              int64 `json:"count"`
	MemoryInBytes      int64 `json:"memory_in_bytes"`
	Hits               int64 `json:"hits"`
	Misses             int64 `json:"misses"`
	Evictions          int64 `json:"evictions"`
	HitsTimeInMillis   int64 `json:"hits_time_in_millis"`
	MissesTimeInMillis int64 `json:"misses_time_in_millis"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool, nodes map[string]elasticsearch.NodeEnrichment) error {
	var data response
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Security Stats API response: %w", err)
	}

	for nodeID, node := range data.Nodes {
		// Skip nodes that returned no DLS data (e.g. nodes without a roles store
		// or older than the feature flag during a rolling upgrade). Reporting an
		// empty event would only add noise.
		if node.Roles == nil || node.Roles.DLS == nil || node.Roles.DLS.Cache == nil {
			continue
		}
		c := node.Roles.DLS.Cache

		event := mb.Event{
			ModuleFields:    mapstr.M{},
			MetricSetFields: mapstr.M{},
		}
		event.ModuleFields.Put("cluster.id", info.ClusterID)
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("node.id", nodeID)
		if enriched, ok := nodes[nodeID]; ok {
			event.ModuleFields.Put("node.name", enriched.Name)
			if len(enriched.Roles) > 0 {
				event.ModuleFields.Put("node.roles", enriched.Roles)
			}
			event.ModuleFields.Put("node.version", enriched.Version)
		}

		// Field names follow the existing thread_pool convention from the
		// node_stats metricset, where every leaf lives under a "<bucket>.count"
		// or "<bucket>.bytes"/"<bucket>.ms" suffix. This avoids name collisions
		// (e.g. a "hits" counter alongside a "hits.time.ms" child path).
		event.MetricSetFields.Put("dls.cache.entries.count", c.Count)
		event.MetricSetFields.Put("dls.cache.memory.bytes", c.MemoryInBytes)
		event.MetricSetFields.Put("dls.cache.hits.count", c.Hits)
		event.MetricSetFields.Put("dls.cache.misses.count", c.Misses)
		event.MetricSetFields.Put("dls.cache.evictions.count", c.Evictions)
		event.MetricSetFields.Put("dls.cache.hits.time.ms", c.HitsTimeInMillis)
		event.MetricSetFields.Put("dls.cache.misses.time.ms", c.MissesTimeInMillis)

		// xpack.enabled in the Metricbeat config used by ECE/ECH writes to
		// .monitoring-* instead of metricbeat-*. Under Elastic Agent the index
		// name is overwritten downstream, so this branch is no-op there.
		if isXpack {
			event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		}

		r.Event(event)
	}

	return nil
}
