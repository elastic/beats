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

package field_stats

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type indexFieldUsage struct {
	Shards []shardFieldUsage `json:"shards"`
}

type shardFieldUsage struct {
	TrackingID              string       `json:"tracking_id"`
	TrackingStartedAtMillis int64        `json:"tracking_started_at_millis"`
	Routing                shardRouting `json:"routing"`
	Stats                  shardStats   `json:"stats"`
}

type shardRouting struct {
	State          string  `json:"state"`
	Primary        bool    `json:"primary"`
	Node           string  `json:"node"`
	RelocatingNode *string `json:"relocating_node"`
}

type shardStats struct {
	AllFields fieldUsage            `json:"all_fields"`
	Fields    map[string]fieldUsage `json:"fields"`
}

type fieldUsage struct {
	Any           int           `json:"any"`
	InvertedIndex invertedIndex `json:"inverted_index"`
	StoredFields  int           `json:"stored_fields"`
	DocValues     int           `json:"doc_values"`
	Points        int           `json:"points"`
	Norms         int           `json:"norms"`
	TermVectors   int           `json:"term_vectors"`
	KnnVectors    int           `json:"knn_vectors"`
}

type invertedIndex struct {
	Terms          int `json:"terms"`
	Postings       int `json:"postings"`
	Proximity      int `json:"proximity"`
	Positions      int `json:"positions"`
	TermFrequencies int `json:"term_frequencies"`
	Offsets        int `json:"offsets"`
	Payloads       int `json:"payloads"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {
	// The response has _shards at the top level, then each index name as a key.
	// We first unmarshal into a raw map to separate _shards from index entries.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(content, &raw); err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Field Usage Stats API response: %w", err)
	}

	var errs []error

	for indexName, indexData := range raw {
		if indexName == "_shards" {
			continue
		}

		var indexUsage indexFieldUsage
		if err := json.Unmarshal(indexData, &indexUsage); err != nil {
			errs = append(errs, fmt.Errorf("failure parsing field usage for index %s: %w", indexName, err))
			continue
		}

		for _, shard := range indexUsage.Shards {
			for fieldName, usage := range shard.Stats.Fields {
				event := mb.Event{}

				event.RootFields = mapstr.M{}
				event.RootFields.Put("service.name", elasticsearch.ModuleName)

				event.ModuleFields = mapstr.M{}
				event.ModuleFields.Put("cluster.name", info.ClusterName)
				event.ModuleFields.Put("cluster.id", info.ClusterID)
				event.ModuleFields.Put("index.name", indexName)

				event.MetricSetFields = mapstr.M{
					"name": fieldName,
					"shard": mapstr.M{
						"tracking_id":               shard.TrackingID,
						"tracking_started_at_millis": shard.TrackingStartedAtMillis,
						"routing": mapstr.M{
							"state":   shard.Routing.State,
							"primary": shard.Routing.Primary,
							"node":    shard.Routing.Node,
						},
					},
					"any":           usage.Any,
					"stored_fields": usage.StoredFields,
					"doc_values":    usage.DocValues,
					"points":        usage.Points,
					"norms":         usage.Norms,
					"term_vectors":  usage.TermVectors,
					"knn_vectors":   usage.KnnVectors,
					"inverted_index": mapstr.M{
						"terms":            usage.InvertedIndex.Terms,
						"postings":         usage.InvertedIndex.Postings,
						"proximity":        usage.InvertedIndex.Proximity,
						"positions":        usage.InvertedIndex.Positions,
						"term_frequencies": usage.InvertedIndex.TermFrequencies,
						"offsets":          usage.InvertedIndex.Offsets,
						"payloads":         usage.InvertedIndex.Payloads,
					},
				}

				if isXpack {
					event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
				}

				r.Event(event)
			}
		}
	}

	return errors.Join(errs...)
}
