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

package index

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	// Based on https://github.com/elastic/elasticsearch/blob/master/x-pack/plugin/monitoring/src/main/java/org/elasticsearch/xpack/monitoring/collector/indices/IndexStatsMonitoringDoc.java#L127-L203
	xpackSchema = s.Schema{
		"uuid":      c.Str("uuid"),
		"primaries": c.Dict("primaries", indexStatsSchema),
		"total":     c.Dict("total", indexStatsSchema),
	}

	indexStatsSchema = s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count": c.Int("count"),
		}),
		"fielddata": c.Dict("fielddata", s.Schema{
			"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
			"evictions":            c.Int("evictions"),
		}),
		"indexing": c.Dict("indexing", s.Schema{
			"index_total":             c.Int("index_total"),
			"index_time_in_millis":    c.Int("index_time_in_millis"),
			"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
		}),
		"merges": c.Dict("merges", s.Schema{
			"total_size_in_bytes": c.Int("total_size_in_bytes"),
		}),
		"query_cache":   c.Dict("query_cache", cacheStatsSchema),
		"request_cache": c.Dict("request_cache", cacheStatsSchema),
		"search": c.Dict("search", s.Schema{
			"query_total":          c.Int("query_total"),
			"query_time_in_millis": c.Int("query_time_in_millis"),
		}),
		"segments": c.Dict("segments", s.Schema{
			"count":                         c.Int("count"),
			"memory_in_bytes":               c.Int("memory_in_bytes"),
			"terms_memory_in_bytes":         c.Int("terms_memory_in_bytes"),
			"stored_fields_memory_in_bytes": c.Int("stored_fields_memory_in_bytes"),
			"term_vectors_memory_in_bytes":  c.Int("term_vectors_memory_in_bytes"),
			"norms_memory_in_bytes":         c.Int("norms_memory_in_bytes"),
			"points_memory_in_bytes":        c.Int("points_memory_in_bytes"),
			"doc_values_memory_in_bytes":    c.Int("doc_values_memory_in_bytes"),
			"index_writer_memory_in_bytes":  c.Int("index_writer_memory_in_bytes"),
			"version_map_memory_in_bytes":   c.Int("version_map_memory_in_bytes"),
			"fixed_bit_set_memory_in_bytes": c.Int("fixed_bit_set_memory_in_bytes"),
		}),
		"store": c.Dict("store", s.Schema{
			"size_in_bytes": c.Int("size_in_bytes"),
		}),
		"refresh": c.Dict("refresh", s.Schema{
			"external_total_time_in_millis": c.Int("external_total_time_in_millis"),
			"total_time_in_millis":          c.Int("total_time_in_millis"),
		}),
	}

	cacheStatsSchema = s.Schema{
		"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
		"evictions":            c.Int("evictions"),
		"hit_count":            c.Int("hit_count"),
		"miss_count":           c.Int("miss_count"),
	}
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var indicesStruct IndicesStruct
	err := json.Unmarshal(content, &indicesStruct)
	if err != nil {
		return errors.Wrap(err, "failure parsing Indices Stats Elasticsearch API response")
	}

	clusterStateMetrics := []string{"metadata", "routing_table"}
	clusterState, err := elasticsearch.GetClusterState(m.HTTP, m.HTTP.GetURI(), clusterStateMetrics)
	if err != nil {
		return errors.Wrap(err, "failure retrieving cluster state from Elasticsearch")
	}

	var errs multierror.Errors
	for name, index := range indicesStruct.Indices {
		event := mb.Event{}
		indexStats, err := xpackSchema.Apply(index)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure applying index stats schema"))
			continue
		}
		indexStats["index"] = name

		err = addClusterStateFields(name, indexStats, clusterState)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure adding cluster state fields"))
			continue
		}

		event.RootFields = common.MapStr{
			"cluster_uuid": info.ClusterID,
			"timestamp":    common.Time(time.Now()),
			"interval_ms":  m.Module().Config().Period / time.Millisecond,
			"type":         "index_stats",
			"index_stats":  indexStats,
		}

		event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		r.Event(event)
	}

	return errs.Err()
}

// Fields added here are based on same fields being added by internal collection in
// https://github.com/elastic/elasticsearch/blob/master/x-pack/plugin/monitoring/src/main/java/org/elasticsearch/xpack/monitoring/collector/indices/IndexStatsMonitoringDoc.java#L62-L124
func addClusterStateFields(indexName string, indexStats, clusterState common.MapStr) error {
	indexMetadata, err := getClusterStateMetricForIndex(clusterState, indexName, "metadata")
	if err != nil {
		return errors.Wrap(err, "failed to get index metadata from cluster state")
	}

	indexRoutingTable, err := getClusterStateMetricForIndex(clusterState, indexName, "routing_table")
	if err != nil {
		return errors.Wrap(err, "failed to get index routing table from cluster state")
	}

	shards, err := getShardsFromRoutingTable(indexRoutingTable)
	if err != nil {
		return errors.Wrap(err, "failed to get shards from routing table")
	}

	created, err := getIndexCreated(indexMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to get index creation time")
	}
	indexStats.Put("created", created)

	// "index_stats.version.created", <--- don't think this is being used in the UI, so can we skip it?
	// "index_stats.version.upgraded", <--- don't think this is being used in the UI, so can we skip it?

	status, err := getIndexStatus(shards)
	if err != nil {
		return errors.Wrap(err, "failed to get index status")
	}
	indexStats.Put("status", status)

	shardStats, err := getIndexShardStats(shards)
	if err != nil {
		return errors.Wrap(err, "failed to get index shard stats")
	}
	indexStats.Put("shards", shardStats)
	return nil
}

func getClusterStateMetricForIndex(clusterState common.MapStr, index, metricKey string) (common.MapStr, error) {
	fieldKey := metricKey + ".indices." + index
	value, err := clusterState.GetValue(fieldKey)
	if err != nil {
		return nil, err
	}

	metric, ok := value.(map[string]interface{})
	if !ok {
		return nil, elastic.MakeErrorForMissingField(fieldKey, elastic.Elasticsearch)
	}
	return common.MapStr(metric), nil
}

func getIndexStatus(shards map[string]interface{}) (string, error) {
	if len(shards) == 0 {
		// No shards, index is red
		return "red", nil
	}

	areAllPrimariesStarted := true
	areAllReplicasStarted := true

	for indexName, indexShard := range shards {
		is, ok := indexShard.([]interface{})
		if !ok {
			return "", fmt.Errorf("shards is not an array")
		}

		for shardIdx, shard := range is {
			s, ok := shard.(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("%v.shards[%v] is not a map", indexName, shardIdx)
			}

			shard := common.MapStr(s)

			isPrimary := shard["primary"].(bool)
			state := shard["state"].(string)

			if isPrimary {
				areAllPrimariesStarted = areAllPrimariesStarted && (state == "STARTED")
			} else {
				areAllReplicasStarted = areAllReplicasStarted && (state == "STARTED")
			}
		}
	}

	if areAllPrimariesStarted && areAllReplicasStarted {
		return "green", nil
	}

	if areAllPrimariesStarted && !areAllReplicasStarted {
		return "yellow", nil
	}

	return "red", nil
}

func getIndexShardStats(shards common.MapStr) (common.MapStr, error) {
	primaries := 0
	replicas := 0

	activePrimaries := 0
	activeReplicas := 0

	unassignedPrimaries := 0
	unassignedReplicas := 0

	initializing := 0
	relocating := 0

	for indexName, indexShard := range shards {
		is, ok := indexShard.([]interface{})
		if !ok {
			return nil, fmt.Errorf("shards is not an array")
		}

		for shardIdx, shard := range is {
			s, ok := shard.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("%v.shards[%v] is not a map", indexName, shardIdx)
			}

			shard := common.MapStr(s)

			isPrimary := shard["primary"].(bool)
			state := shard["state"].(string)

			if isPrimary {
				primaries++
				switch state {
				case "STARTED":
					activePrimaries++
				case "UNASSIGNED":
					unassignedPrimaries++
				}
			} else {
				replicas++
				switch state {
				case "STARTED":
					activeReplicas++
				case "UNASSIGNED":
					unassignedReplicas++
				}
			}

			switch state {
			case "INITIALIZING":
				initializing++
			case "RELOCATING":
				relocating++
			}
		}
	}

	return common.MapStr{
		"total":     primaries + replicas,
		"primaries": primaries,
		"replicas":  replicas,

		"active_total":     activePrimaries + activeReplicas,
		"active_primaries": activePrimaries,
		"active_replicas":  activeReplicas,

		"unassigned_total":     unassignedPrimaries + unassignedReplicas,
		"unassigned_primaries": unassignedPrimaries,
		"unassigned_replicas":  unassignedReplicas,

		"initializing": initializing,
		"relocating":   relocating,
	}, nil
}

func getIndexCreated(indexMetadata common.MapStr) (int64, error) {
	v, err := indexMetadata.GetValue("settings.index.creation_date")
	if err != nil {
		return 0, err
	}

	c, ok := v.(string)
	if !ok {
		return 0, elastic.MakeErrorForMissingField("settings.index.creation_date", elastic.Elasticsearch)
	}

	return strconv.ParseInt(c, 10, 64)
}

func getShardsFromRoutingTable(indexRoutingTable common.MapStr) (map[string]interface{}, error) {
	s, err := indexRoutingTable.GetValue("shards")
	if err != nil {
		return nil, err
	}

	shards, ok := s.(map[string]interface{})
	if !ok {
		return nil, elastic.MakeErrorForMissingField("shards", elastic.Elasticsearch)
	}

	return shards, nil
}
