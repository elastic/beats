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

	"github.com/elastic/beats/v7/libbeat/common"
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

func eventMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {
	var all struct {
		Data map[string]interface{} `json:"_all"`
	}

	err := json.Unmarshal(content, &all)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Stats API response: %v", err)
	}

	fields, err := schema.Apply(all.Data, s.FailOnRequired)
	if err != nil {
		return fmt.Errorf("failure applying stats schema: %v", err)
	}

	var event mb.Event
	event.RootFields = common.MapStr{}
	_, _ = event.RootFields.Put("service.name", elasticsearch.ModuleName)

	event.ModuleFields = common.MapStr{}
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
