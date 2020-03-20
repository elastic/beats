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
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	errParse = errors.New("failure parsing Indices Stats Elasticsearch API response")
)

// Based on https://github.com/elastic/elasticsearch/blob/master/x-pack/plugin/monitoring/src/main/java/org/elasticsearch/xpack/monitoring/collector/indices/IndexStatsMonitoringDoc.java#L127-L203
type stats struct {
	Indices map[string]index `json:"indices"`
}

type index struct {
	UUID      string     `json:"uuid"`
	Primaries indexStats `json:"primaries"`
	Total     indexStats `json:"total"`

	Index   string     `json:"index"`
	Created int64      `json:"created"`
	Status  string     `json:"status"`
	Shards  shardStats `json:"shards"`
}

type indexStats struct {
	Docs struct {
		Count int `json:"count"`
	} `json:"docs"`
	FieldData struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		Evictions         int `json:"evictions"`
	} `json:"fielddata"`
	Indexing struct {
		IndexTotal           int `json:"index_total"`
		IndexTimeInMillis    int `json:"index_time_in_millis"`
		ThrottleTimeInMillis int `json:"throttle_time_in_millis"`
	} `json:"indexing"`
	Merges struct {
		TotalSizeInBytes int `json:"total_size_in_bytes"`
	} `json:"merges"`
	QueryCache   cacheStats `json:"query_cache"`
	RequestCache cacheStats `json:"request_cache"`
	Search       struct {
		QueryTotal        int `json:"query_total"`
		QueryTimeInMillis int `json:"query_time_in_millis"`
	} `json:"search"`
	Segments struct {
		Count                     int `json:"count"`
		MemoryInBytes             int `json:"memory_in_bytes"`
		TermsMemoryInBytes        int `json:"terms_memory_in_bytes"`
		StoredFieldsMemoryInBytes int `json:"stored_fields_memory_in_bytes"`
		TermVectorsMemoryInBytes  int `json:"term_vectors_memory_in_bytes"`
		NormsMemoryInBytes        int `json:"norms_memory_in_bytes"`
		PointsMemoryInBytes       int `json:"points_memory_in_bytes"`
		DocValuesMemoryInBytes    int `json:"doc_values_memory_in_bytes"`
		IndexWriterMemoryInBytes  int `json:"index_writer_memory_in_bytes"`
		VersionMapMemoryInBytes   int `json:"version_map_memory_in_bytes"`
		FixedBitSetMemoryInBytes  int `json:"fixed_bit_set_memory_in_bytes"`
	} `json:"segments"`
	Store struct {
		SizeInBytes int `json:"size_in_bytes"`
	} `json:"store"`
	Refresh struct {
		ExternalTotalTimeInMillis int `json:"external_total_time_in_millis"`
		TotalTimeInMillis         int `json:"total_time_in_millis"`
	} `json:"refresh"`
}

type cacheStats struct {
	MemorySizeInBytes int `json:"memory_size_in_bytes"`
	Evictions         int `json:"evictions"`
	HitCount          int `json:"hit_count"`
	MissCount         int `json:"miss_count"`
}

type shardStats struct {
	Total     int `json:"total"`
	Primaries int `json:"primaries"`
	Replicas  int `json:"replicas"`

	ActiveTotal     int `json:"active_total"`
	ActivePrimaries int `json:"active_primaries"`
	ActiveReplicas  int `json:"active_replicas"`

	UnassignedTotal     int `json:"unassigned_total"`
	UnassignedPrimaries int `json:"unassigned_primaries"`
	UnassignedReplicas  int `json:"unassigned_replicas"`

	Initializing int `json:"initializing"`
	Relocating   int `json:"relocating"`
}

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	clusterStateMetrics := []string{"metadata", "routing_table"}
	clusterState, err := elasticsearch.GetClusterState(m.HTTP, m.HTTP.GetURI(), clusterStateMetrics)
	if err != nil {
		return errors.Wrap(err, "failure retrieving cluster state from Elasticsearch")
	}

	var indicesStats stats
	if err := parseAPIResponse(content, &indicesStats); err != nil {
		return errors.Wrap(err, "failure parsing Indices Stats Elasticsearch API response")
	}

	var errs multierror.Errors
	for name, idx := range indicesStats.Indices {
		event := mb.Event{}
		idx.Index = name

		err = addClusterStateFields(&idx, clusterState)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure adding cluster state fields"))
			continue
		}

		event.RootFields = common.MapStr{
			"cluster_uuid": info.ClusterID,
			"timestamp":    common.Time(time.Now()),
			"interval_ms":  m.Module().Config().Period / time.Millisecond,
			"type":         "index_stats",
			"index_stats":  idx,
		}

		event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		r.Event(event)
	}

	return errs.Err()
}

func parseAPIResponse(content []byte, indicesStats *stats) error {
	return json.Unmarshal(content, indicesStats)
}

// Fields added here are based on same fields being added by internal collection in
// https://github.com/elastic/elasticsearch/blob/master/x-pack/plugin/monitoring/src/main/java/org/elasticsearch/xpack/monitoring/collector/indices/IndexStatsMonitoringDoc.java#L62-L124
func addClusterStateFields(idx *index, clusterState common.MapStr) error {
	indexMetadata, err := getClusterStateMetricForIndex(clusterState, idx.Index, "metadata")
	if err != nil {
		return errors.Wrap(err, "failed to get index metadata from cluster state")
	}

	indexRoutingTable, err := getClusterStateMetricForIndex(clusterState, idx.Index, "routing_table")
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
	idx.Created = created

	// "index_stats.version.created", <--- don't think this is being used in the UI, so can we skip it?
	// "index_stats.version.upgraded", <--- don't think this is being used in the UI, so can we skip it?

	status, err := getIndexStatus(shards)
	if err != nil {
		return errors.Wrap(err, "failed to get index status")
	}
	idx.Status = status

	shardStats, err := getIndexShardStats(shards)
	if err != nil {
		return errors.Wrap(err, "failed to get index shard stats")
	}
	idx.Shards = *shardStats
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

func getIndexShardStats(shards common.MapStr) (*shardStats, error) {
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

	return &shardStats{
		Total:               primaries + replicas,
		Primaries:           primaries,
		Replicas:            replicas,
		ActiveTotal:         activePrimaries + activeReplicas,
		ActivePrimaries:     activePrimaries,
		ActiveReplicas:      activeReplicas,
		UnassignedTotal:     unassignedPrimaries + unassignedReplicas,
		UnassignedPrimaries: unassignedPrimaries,
		UnassignedReplicas:  unassignedReplicas,
		Initializing:        initializing,
		Relocating:          relocating,
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
