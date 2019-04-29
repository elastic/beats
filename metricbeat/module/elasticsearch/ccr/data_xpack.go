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

package ccr

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch CCR Stats API response")
	}

	now := common.Time(time.Now())
	intervalMS := m.Module().Config().Period / time.Millisecond
	index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)

	indexCCRStats(r, data, info, now, intervalMS, index)
	indexCCRAutoFollowStats(r, data, info, now, intervalMS, index)
	return nil
}

func indexCCRStats(r mb.ReporterV2, ccrData response, esInfo elasticsearch.Info, now common.Time, intervalMS time.Duration, indexName string) {
	for _, followerIndex := range ccrData.FollowStats.Indices {
		for _, followerShard := range followerIndex.Shards {
			event := mb.Event{}
			event.RootFields = common.MapStr{
				"cluster_uuid": esInfo.ClusterID,
				"timestamp":    now,
				"interval_ms":  intervalMS,
				"type":         "ccr_stats",
				"ccr_stats":    followerShard,
			}

			event.Index = indexName
			r.Event(event)
		}
	}
}

func indexCCRAutoFollowStats(r mb.ReporterV2, ccrData response, esInfo elasticsearch.Info, now common.Time, intervalMS time.Duration, indexName string) {
	event := mb.Event{}
	event.RootFields = common.MapStr{
		"cluster_uuid":          esInfo.ClusterID,
		"timestamp":             now,
		"interval_ms":           intervalMS,
		"type":                  "ccr_auto_follow_stats",
		"ccr_auto_follow_stats": ccrData.AutoFollowStats,
	}

	event.Index = indexName
	r.Event(event)
}
