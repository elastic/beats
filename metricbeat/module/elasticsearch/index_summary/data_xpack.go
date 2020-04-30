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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	xpackSchema = s.Schema{
		"primaries": c.Dict("primaries", indexStatsSchema),
		"total":     c.Dict("total", indexStatsSchema),
	}

	indexStatsSchema = s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count": c.Int("count"),
		}),
		"store": c.Dict("store", s.Schema{
			"size_in_bytes": c.Int("size_in_bytes"),
		}),
		"indexing": c.Dict("indexing", s.Schema{
			"index_total":             c.Int("index_total"),
			"index_time_in_millis":    c.Int("index_time_in_millis"),
			"is_throttled":            c.Bool("is_throttled"),
			"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
		}),
		"bulk": elasticsearch.BulkStatsDict,
		"search": c.Dict("search", s.Schema{
			"query_total":          c.Int("query_total"),
			"query_time_in_millis": c.Int("query_time_in_millis"),
		}),
	}
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var all struct {
		Data map[string]interface{} `json:"_all"`
	}

	err := json.Unmarshal(content, &all)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Stats API response")
	}

	p := all.Data["primaries"]
	primaries, ok := p.(map[string]interface{})
	if !ok {
		return fmt.Errorf("primaries is not a map")
	}

	if len(primaries) == 0 {
		// There is no data in the cluster, hence no metrics to parse or report
		return nil
	}

	fields, err := xpackSchema.Apply(all.Data)
	if err != nil {
		return errors.Wrap(err, "failure applying stats schema")
	}

	event := mb.Event{}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("indices_stats._all", fields)
	event.RootFields.Put("cluster_uuid", info.ClusterID)
	event.RootFields.Put("timestamp", common.Time(time.Now()))
	event.RootFields.Put("interval_ms", m.Module().Config().Period/time.Millisecond)
	event.RootFields.Put("type", "indices_stats")

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)

	r.Event(event)
	return nil
}
