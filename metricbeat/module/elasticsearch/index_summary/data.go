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

type nodeStatsWrapper struct {
	Nodes map[string]interface{} `json:"nodes"`
}

var nodeItemSchema = s.Schema{
	"indices": c.Dict("indices", s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count":   c.Int("count", s.Required),
			"deleted": c.Int("deleted", s.Required),
		}, c.DictRequired),
		"store": c.Dict("store", s.Schema{
			"size": s.Object{
				"bytes": c.Int("size_in_bytes", s.Required),
			},
			"total_data_set_size": s.Object{
				"bytes": c.Int("total_data_set_size_in_bytes", s.Required),
			},
		}, c.DictRequired),
		"indexing": c.Dict("indexing", s.Schema{
			"index": s.Object{
				"count": c.Int("index_total", s.Required),
				"time": s.Object{
					"ms": c.Int("index_time_in_millis", s.Required),
				},
			},
		}, c.DictRequired),
		"search": c.Dict("search", s.Schema{
			"query": s.Object{
				"count": c.Int("query_total", s.Required),
				"time": s.Object{
					"ms": c.Int("query_time_in_millis", s.Required),
				},
			},
		}, c.DictRequired),
		"segments": c.Dict("segments", s.Schema{
			"count": c.Int("count", s.Required),
			"memory": s.Object{
				"bytes": c.Int("memory_in_bytes", s.Required),
			},
		}, c.DictRequired),
		"bulk": c.Dict("bulk", s.Schema{
			"operations": s.Object{
				"count": c.Int("total_operations", s.Required),
			},
			"time": s.Object{
				"avg": s.Object{
					"bytes": c.Int("avg_size_in_bytes", s.Required),
				},
			},
			"size": s.Object{
				"bytes": c.Int("total_size_in_bytes", s.Required),
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

func eventMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXPack bool) error {
	var wrapper nodeStatsWrapper
	if err := json.Unmarshal(content, &wrapper); err != nil {
		return fmt.Errorf("failure parsing NodeStats API response: %w", err)
	}

	if len(wrapper.Nodes) == 0 {
		return fmt.Errorf("no nodes found in NodeStats response")
	}

	var total IndexSummary
	for nodeKey, raw := range wrapper.Nodes {
		err := addNodeMetrics(raw, &total)
		if err != nil {
			return fmt.Errorf("error processing node %q: %w", nodeKey, err)
		}
	}

	event := buildEvent(&info, &total, isXPack)
	r.Event(event)
	return nil
}

func addNodeMetrics(rawNode interface{}, summary *IndexSummary) error {
	nodeMap, ok := rawNode.(map[string]interface{})
	if !ok {
		return fmt.Errorf("node is not a map")
	}

	validated, err := nodeItemSchema.Apply(nodeMap, s.FailOnRequired)
	if err != nil {
		return err
	}

	incrementValue := func(dst *int64, path ...string) {
		if v, err := getInt64(validated, path...); err == nil {
			*dst += v
		}
	}

	// Docs
	incrementValue(&summary.Docs.Count, "indices", "docs", "count")
	incrementValue(&summary.Docs.Deleted, "indices", "docs", "deleted")

	// Store
	incrementValue(&summary.Store.Size.Bytes, "indices", "store", "size", "bytes")
	incrementValue(&summary.Store.TotalDataSetSize.Bytes, "indices", "store", "total_data_set_size", "bytes")

	// Indexing
	incrementValue(&summary.Indexing.Index.Count, "indices", "indexing", "index", "count")
	incrementValue(&summary.Indexing.Index.Time.Ms, "indices", "indexing", "index", "time", "ms")

	// Search
	incrementValue(&summary.Search.Query.Count, "indices", "search", "query", "count")
	incrementValue(&summary.Search.Query.Time.Ms, "indices", "search", "query", "time", "ms")

	// Segments
	incrementValue(&summary.Segments.Count, "indices", "segments", "count")
	incrementValue(&summary.Segments.Memory.Bytes, "indices", "segments", "memory", "bytes")

	return nil
}

func getInt64(m mapstr.M, path ...string) (int64, error) {
	current := interface{}(m)
	for _, key := range path {
		mm, ok := current.(mapstr.M)
		if !ok {
			return 0, fmt.Errorf("expected mapstr.M at %q, got %T", key, current)
		}
		val, ok := mm[key]
		if !ok {
			return 0, fmt.Errorf("missing key: %q", key)
		}
		current = val
	}

	i, ok := current.(int64)
	if !ok {
		return 0, fmt.Errorf("expected int64 at path %v, got %T", path, current)
	}
	return i, nil
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
