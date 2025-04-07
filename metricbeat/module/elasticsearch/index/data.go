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

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

type IndicesStruct struct {
	Indices map[string]map[string]interface{} `json:"indices"`
}

<<<<<<< HEAD
var (
	schema = s.Schema{
		"total": c.Dict("total", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{
					"bytes": c.Int("size_in_bytes"),
				},
			}),
			"segments": c.Dict("segments", s.Schema{
				"count": c.Int("count"),
				"memory": s.Object{
					"bytes": c.Int("memory_in_bytes"),
				},
			}),
		}),
=======
type Index struct {
	UUID      string    `json:"uuid"`
	Primaries primaries `json:"primaries"`
	Total     total     `json:"total"`

	Index          string     `json:"index"`
	Status         string     `json:"status"`
	TierPreference string     `json:"tier_preference,omitempty"`
	CreationDate   string     `json:"creation_date,omitempty"`
	Version        string     `json:"version,omitempty"`
	Shards         shardStats `json:"shards"`
}

type primaries struct {
	Docs struct {
		Count   int `json:"count"`
		Deleted int `json:"deleted"`
	} `json:"docs"`
	Indexing struct {
		IndexTotal           int `json:"index_total"`
		IndexTimeInMillis    int `json:"index_time_in_millis"`
		ThrottleTimeInMillis int `json:"throttle_time_in_millis"`
	} `json:"indexing"`
	Merges struct {
		TotalSizeInBytes int `json:"total_size_in_bytes"`
	} `json:"merges"`
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
		SizeInBytes             int `json:"size_in_bytes"`
		TotalDataSetSizeInBytes int `json:"total_data_set_size_in_bytes"`
	} `json:"store"`
	Refresh struct {
		TotalTimeInMillis         int `json:"total_time_in_millis"`
		ExternalTotalTimeInMillis int `json:"external_total_time_in_millis"`
	} `json:"refresh"`
	QueryCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
	} `json:"query_cache"`
	RequestCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
		Evictions         int `json:"evictions"`
	} `json:"request_cache"`
	Search struct {
		QueryTotal        int `json:"query_total"`
		QueryTimeInMillis int `json:"query_time_in_millis"`
	} `json:"search"`
}

type total struct {
	Docs struct {
		Count   int `json:"count"`
		Deleted int `json:"deleted"`
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
	Bulk   *bulkStats `json:"bulk,omitempty"`
	Merges struct {
		TotalSizeInBytes int `json:"total_size_in_bytes"`
	} `json:"merges"`
	Search struct {
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
		SizeInBytes             int `json:"size_in_bytes"`
		TotalDataSetSizeInBytes int `json:"total_data_set_size_in_bytes"`
	} `json:"store"`
	Refresh struct {
		TotalTimeInMillis         int `json:"total_time_in_millis"`
		ExternalTotalTimeInMillis int `json:"external_total_time_in_millis"`
	} `json:"refresh"`
	QueryCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
		Evictions         int `json:"evictions"`
	} `json:"query_cache"`
	RequestCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
		Evictions         int `json:"evictions"`
	} `json:"request_cache"`
}

type shardStats struct {
	Total     int `json:"total"`
	Primaries int `json:"primaries"`
	Replicas  int `json:"-"`

	ActiveTotal     int `json:"-"`
	ActivePrimaries int `json:"-"`
	ActiveReplicas  int `json:"-"`

	UnassignedTotal     int `json:"-"`
	UnassignedPrimaries int `json:"-"`
	UnassignedReplicas  int `json:"-"`

	Initializing int `json:"-"`
	Relocating   int `json:"-"`
}

type bulkStats struct {
	TotalOperations   int `json:"total_operations"`
	TotalTimeInMillis int `json:"total_time_in_millis"`
	TotalSizeInBytes  int `json:"total_size_in_bytes"`
	AvgTimeInMillis   int `json:"avg_time_in_millis"`
	AvgSizeInBytes    int `json:"avg_size_in_bytes"`
}

var logger = logp.NewLogger("elasticsearch.index")

func eventsMapping(r mb.ReporterV2, httpClient *helper.HTTP, info elasticsearch.Info, content []byte, isXpack bool) error {
	clusterStateMetrics := []string{"routing_table"}
	clusterStateFilterPaths := []string{"routing_table"}
	clusterState, err := elasticsearch.GetClusterState(httpClient, httpClient.GetURI(), clusterStateMetrics, clusterStateFilterPaths)
	if err != nil {
		return fmt.Errorf("failure retrieving cluster state from Elasticsearch: %w", err)
>>>>>>> 999fcb65b (Added omitempty on new elasticsearch package fields (#43637))
	}
)

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	var indicesStruct IndicesStruct
	err := json.Unmarshal(content, &indicesStruct)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Stats API response")
	}

	var errs multierror.Errors
	for name, index := range indicesStruct.Indices {
		event := mb.Event{}

		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)

		event.MetricSetFields, err = schema.Apply(index)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure applying index schema"))
			continue
		}
		// Write name here as full name only available as key
		event.MetricSetFields["name"] = name
		r.Event(event)
	}

	return errs.Err()
}
