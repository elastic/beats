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

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

type indicesResponse map[string]common.MapStr

var schema = s.Schema{
	"avg_item_size":        c.Int("avg_item_size"),
	"avg_scan_latency":     c.Int("avg_scan_latency"),
	"cache_hits":           c.Int("cache_hits"),
	"cache_misses":         c.Int("cache_misses"),
	"data_size":            c.Int("data_size"),
	"disk_size":            c.Int("disk_size"),
	"frag_percent":         c.Int("frag_percent"),
	"items_count":          c.Int("items_count"),
	"memory_used":          c.Int("memory_used"),
	"num_docs_indexed":     c.Int("num_docs_indexed"),
	"num_docs_pending":     c.Int("num_docs_pending"),
	"num_docs_queued":      c.Int("num_docs_queued"),
	"num_items_flushed":    c.Int("num_items_flushed"),
	"num_pending_requests": c.Int("num_pending_requests"),
	"num_requests":         c.Int("num_requests"),
	"num_scan_errors":      c.Int("num_scan_errors"),
	"num_scan_timeouts":    c.Int("num_scan_timeouts"),
	"resident_percent":     c.Int("resident_percent"),
}

func eventsMapping(r mb.ReporterV2, content []byte) error {
	var ir indicesResponse
	err := json.Unmarshal(content, &ir)
	if err != nil {
		return err
	}

	for indexName, index := range ir {
		if indexName == "indexer" {
			continue
		}

		indexMapstr, _ := schema.Apply(index)
		indexMapstr["name"] = indexName

		r.Event(mb.Event{
			MetricSetFields: indexMapstr,
		})
	}

	return nil
}
