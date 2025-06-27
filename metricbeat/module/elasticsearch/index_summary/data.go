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

package index_summary

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"primaries": c.Dict("primaries", indexSummaryDict),
		"total":     c.Dict("total", indexSummaryDict),
	}
)

var indexSummaryDict = s.Schema{
	"docs": c.Dict("docs", s.Schema{
		"count":   c.Int("count"),
		"deleted": c.Int("deleted"),
	}),
	"store": c.Dict("store", s.Schema{
		"size": s.Object{
			"bytes": c.Int("size_in_bytes"),
		},
		"total_data_set_size": s.Object{
			"bytes": c.Int("total_data_set_size_in_bytes", s.Optional),
		},
	}),
	"segments": c.Dict("segments", s.Schema{
		"count": c.Int("count"),
		"memory": s.Object{
			"bytes": c.Int("memory_in_bytes"),
		},
	}),
	"indexing": indexingDict,
	"bulk":     bulkStatsDict,
	"search":   searchDict,
}

var indexingDict = c.Dict("indexing", s.Schema{
	"index": s.Object{
		"count": c.Int("index_total"),
		"time": s.Object{
			"ms": c.Int("index_time_in_millis"),
		},
	},
})

var searchDict = c.Dict("search", s.Schema{
	"query": s.Object{
		"count": c.Int("query_total"),
		"time": s.Object{
			"ms": c.Int("query_time_in_millis"),
		},
	},
})

var bulkStatsDict = c.Dict("bulk", s.Schema{
	"operations": s.Object{
		"count": c.Int("total_operations"),
	},
	"time": s.Object{
		"avg": s.Object{
			"bytes": c.Int("avg_size_in_bytes"),
		},
	},
	"size": s.Object{
		"bytes": c.Int("total_size_in_bytes"),
	},
}, c.DictOptional)

type nodeStatsWrapper struct {
	Nodes map[string]interface{} `json:"nodes"`
}

var nodeItemSchema = s.Schema{
	"indices": c.Dict("indices", s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count":   c.Int("count"),
			"deleted": c.Int("deleted"),
		}),
		"store": c.Dict("store", s.Schema{
			"size": s.Object{
				"bytes": c.Int("size_in_bytes"),
			},
			"total_data_set_size": s.Object{
				"bytes": c.Int("total_data_set_size_in_bytes"),
			},
		}),
		"indexing": c.Dict("indexing", s.Schema{
			"index": s.Object{
				"count": c.Int("index_total"),
				"time": s.Object{
					"ms": c.Int("index_time_in_millis"),
				},
			},
		}),
		"search": c.Dict("search", s.Schema{
			"query": s.Object{
				"count": c.Int("query_total"),
				"time": s.Object{
					"ms": c.Int("query_time_in_millis"),
				},
			},
		}),
		"segments": c.Dict("segments", s.Schema{
			"count": c.Int("count"),
			"memory": s.Object{
				"bytes": c.Int("memory_in_bytes"),
			},
		}),
		"bulk": c.Dict("bulk", s.Schema{
			"operations": s.Object{
				"count": c.Int("total_operations"),
			},
			"time": s.Object{
				"avg": s.Object{
					"bytes": c.Int("avg_size_in_bytes"),
				},
			},
			"size": s.Object{
				"bytes": c.Int("total_size_in_bytes"),
			},
		}, c.DictOptional),
	}),
}

type IndexSummaryMetricSet struct {
	Primaries IndexSummary `json:"primaries"`
	Total     IndexSummary `json:"total"`
}

type IndexSummary struct {
	Docs     DocsSection     `json:"docs"`
	Store    StoreSection    `json:"store"`
	Indexing IndexingSection `json:"indexing"`
	Search   SearchSection   `json:"search"`
	Segments SegmentSection  `json:"segments"`
	Bulk     BulkSection     `json:"bulk"`
}

type DocsSection struct {
	Count   int64 `json:"count"`
	Deleted int64 `json:"deleted"`
}

type StoreSection struct {
	Size struct {
		Bytes int64 `json:"bytes"`
	} `json:"size"`
	TotalDataSetSize struct {
		Bytes int64 `json:"bytes"`
	} `json:"total_data_set_size"`
}

type IndexingSection struct {
	Index struct {
		Count int64 `json:"count"`
		Time  struct {
			Ms int64 `json:"ms"`
		} `json:"time"`
	} `json:"index"`
}

type SearchSection struct {
	Query struct {
		Count int64 `json:"count"`
		Time  struct {
			Ms int64 `json:"ms"`
		} `json:"time"`
	} `json:"query"`
}

type SegmentSection struct {
	Count  int64 `json:"count"`
	Memory struct {
		Bytes int64 `json:"bytes"`
	} `json:"memory"`
}

type BulkSection struct {
	Operations struct {
		Count int64 `json:"count"`
	} `json:"operations"`
	Time struct {
		Avg struct {
			Bytes int64 `json:"bytes"`
		} `json:"avg"`
	} `json:"time"`
	Size struct {
		Bytes int64 `json:"bytes"`
	} `json:"size"`
}

func eventMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {
	var all struct {
		Data map[string]interface{} `json:"_all"`
	}

	err := json.Unmarshal(content, &all)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Stats API response: %w", err)
	}

	fields, err := schema.Apply(all.Data, s.FailOnRequired)
	if err != nil {
		return fmt.Errorf("failure applying stats schema: %w", err)
	}

	var event mb.Event
	event.RootFields = mapstr.M{}
	_, _ = event.RootFields.Put("service.name", elasticsearch.ModuleName)

	event.ModuleFields = mapstr.M{}
	_, _ = event.ModuleFields.Put("cluster.name", info.ClusterName)
	_, _ = event.ModuleFields.Put("cluster.id", info.ClusterID)

	event.MetricSetFields = fields

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		event.Index = index
	}

	r.Event(event)

	return nil
}

func eventMappingNewEndpoint(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXPack bool) error {
	var wrapper nodeStatsWrapper
	if err := json.Unmarshal(content, &wrapper); err != nil {
		return fmt.Errorf("failure parsing NodeStats API response: %w", err)
	}

	if len(wrapper.Nodes) == 0 {
		return fmt.Errorf("no nodes found in NodeStats response")
	}

	var total IndexSummary
	for nodeKey, raw := range wrapper.Nodes {
		summary, err := processNode(raw)
		if err != nil {
			return fmt.Errorf("error processing node %q: %w", nodeKey, err)
		}
		total.merge(&summary)
	}

	event := buildEvent(&info, &total, isXPack)
	r.Event(event)
	return nil
}

func processNode(rawNode interface{}) (IndexSummary, error) {
	var summary IndexSummary

	nodeMap, ok := rawNode.(map[string]interface{})
	if !ok {
		return summary, fmt.Errorf("node is not a map")
	}

	validated, err := nodeItemSchema.Apply(nodeMap, s.FailOnRequired)
	if err != nil {
		return summary, err
	}

	getM := func(m mapstr.M, key string) mapstr.M {
		return m[key].(mapstr.M)
	}

	indices := getM(validated, "indices")
	docs := getM(indices, "docs")
	store := getM(indices, "store")
	indexing := getM(indices, "indexing")
	search := getM(indices, "search")
	segments := getM(indices, "segments")

	// Docs
	summary.Docs.Count = docs["count"].(int64)
	summary.Docs.Deleted = docs["deleted"].(int64)

	// Store
	summary.Store.Size.Bytes = getM(store, "size")["bytes"].(int64)
	if tds, ok := store["total_data_set_size"].(mapstr.M); ok {
		summary.Store.TotalDataSetSize.Bytes = tds["bytes"].(int64)
	}

	// Indexing
	index := getM(indexing, "index")
	summary.Indexing.Index.Count = index["count"].(int64)
	summary.Indexing.Index.Time.Ms = getM(index, "time")["ms"].(int64)

	// Search
	query := getM(getM(search, "query"), "time")
	summary.Search.Query.Count = getM(search, "query")["count"].(int64)
	summary.Search.Query.Time.Ms = query["ms"].(int64)

	// Segments
	summary.Segments.Count = segments["count"].(int64)
	summary.Segments.Memory.Bytes = getM(segments, "memory")["bytes"].(int64)

	// Bulk (optional)
	if bulkRaw, ok := indices["bulk"].(mapstr.M); ok {
		ops := getM(bulkRaw, "operations")
		time := getM(getM(bulkRaw, "time"), "avg")
		size := getM(bulkRaw, "size")

		summary.Bulk.Operations.Count = ops["count"].(int64)
		summary.Bulk.Size.Bytes = size["bytes"].(int64)

		summary.Bulk.Time.Avg.Bytes = time["bytes"].(int64)

	}

	return summary, nil
}

func (dst *IndexSummary) merge(src *IndexSummary) {
	dst.Docs.Count += src.Docs.Count
	dst.Docs.Deleted += src.Docs.Deleted

	dst.Store.Size.Bytes += src.Store.Size.Bytes
	dst.Store.TotalDataSetSize.Bytes += src.Store.TotalDataSetSize.Bytes

	dst.Indexing.Index.Count += src.Indexing.Index.Count
	dst.Indexing.Index.Time.Ms += src.Indexing.Index.Time.Ms

	dst.Search.Query.Count += src.Search.Query.Count
	dst.Search.Query.Time.Ms += src.Search.Query.Time.Ms

	dst.Segments.Count += src.Segments.Count
	dst.Segments.Memory.Bytes += src.Segments.Memory.Bytes

	dst.Bulk.Operations.Count += src.Bulk.Operations.Count
	dst.Bulk.Size.Bytes += src.Bulk.Size.Bytes
	dst.Bulk.Time.Avg.Bytes += src.Bulk.Time.Avg.Bytes
}

func buildEvent(info *elasticsearch.Info, summary *IndexSummary, isXPack bool) mb.Event {
	eventNew := map[string]interface{}{
		"primaries": summary,
		"total":     summary,
	}

	var event mb.Event
	event.RootFields = mapstr.M{}
	_, _ = event.RootFields.Put("service.name", elasticsearch.ModuleName)

	event.ModuleFields = mapstr.M{}
	_, _ = event.ModuleFields.Put("cluster.name", info.ClusterName)
	_, _ = event.ModuleFields.Put("cluster.id", info.ClusterID)

	event.MetricSetFields = eventNew

	if isXPack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		event.Index = index
	}

	return event
}

func eventMappingNewEndpoint2(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXPack bool) error {
	var nodesWrapper nodeStatsWrapper

	err := json.Unmarshal(content, &nodesWrapper)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch NodeStats API response: %w", err)
	}

	if len(nodesWrapper.Nodes) == 0 {
		return fmt.Errorf("no nodes found in NodeStats response")
	}

	var aggregations IndexSummary

	for _, rawNode := range nodesWrapper.Nodes {
		validated, err := nodeItemSchema.Apply(rawNode.(map[string]interface{}), s.FailOnRequired)
		if err != nil {
			return fmt.Errorf("schema validation failed for node: %w", err)
		}

		indices := validated["indices"].(mapstr.M)

		// Docs
		docs := indices["docs"].(mapstr.M)
		aggregations.Docs.Count += docs["count"].(int64)
		aggregations.Docs.Deleted += docs["deleted"].(int64)

		// Store
		store := indices["store"].(mapstr.M)
		size := store["size"].(mapstr.M)
		aggregations.Store.Size.Bytes += size["bytes"].(int64)

		if tds, ok := store["total_data_set_size"].(mapstr.M); ok {
			aggregations.Store.TotalDataSetSize.Bytes += tds["bytes"].(int64)
		}

		// Indexing
		indexing := indices["indexing"].(mapstr.M)
		index := indexing["index"].(mapstr.M)
		aggregations.Indexing.Index.Count += index["count"].(int64)
		aggregations.Indexing.Index.Time.Ms += index["time"].(mapstr.M)["ms"].(int64)

		// Search
		search := indices["search"].(mapstr.M)
		query := search["query"].(mapstr.M)
		aggregations.Search.Query.Count += query["count"].(int64)
		aggregations.Search.Query.Time.Ms += query["time"].(mapstr.M)["ms"].(int64)

		// Segments
		segments := indices["segments"].(mapstr.M)
		aggregations.Segments.Count += segments["count"].(int64)
		aggregations.Segments.Memory.Bytes += segments["memory"].(mapstr.M)["bytes"].(int64)

		// Bulk (optional)
		bulkRaw, exists := indices["bulk"].(mapstr.M)
		if exists {
			aggregations.Bulk.Operations.Count += bulkRaw["operations"].(mapstr.M)["count"].(int64)

			bulkTime := bulkRaw["time"].(mapstr.M)
			aggregations.Bulk.Time.Avg.Bytes += bulkTime["avg"].(mapstr.M)["bytes"].(int64)

			bulkSize := bulkRaw["size"].(mapstr.M)
			aggregations.Bulk.Size.Bytes += bulkSize["bytes"].(int64)
		}
	}

	eventNew := map[string]interface{}{
		"primaries": aggregations,
		"total":     aggregations,
	}

	var event mb.Event
	event.RootFields = mapstr.M{}
	_, _ = event.RootFields.Put("service.name", elasticsearch.ModuleName)

	event.ModuleFields = mapstr.M{}
	_, _ = event.ModuleFields.Put("cluster.name", info.ClusterName)
	_, _ = event.ModuleFields.Put("cluster.id", info.ClusterID)

	event.MetricSetFields = eventNew

	if isXPack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		event.Index = index
	}

	r.Event(event)

	return nil
}
