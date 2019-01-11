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

package index_recovery

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Recovery API response")
	}

	var results []map[string]interface{}
	for indexName, indexData := range data {
		indexData, ok := indexData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%v is not a map", indexName)
		}

		shards, ok := indexData["shards"]
		if !ok {
			return elastic.MakeErrorForMissingField(indexName+".shards", elastic.Elasticsearch)
		}

		shardsArr, ok := shards.([]interface{})
		if !ok {
			return fmt.Errorf("%v.shards is not an array", indexName)
		}

		for shardIdx, shard := range shardsArr {
			shard, ok := shard.(map[string]interface{})
			if !ok {
				return fmt.Errorf("%v.shards[%v] is not a map", indexName, shardIdx)
			}

			shard["index_name"] = indexName
			results = append(results, shard)
		}
	}

	indexRecovery := common.MapStr{}
	indexRecovery["shards"] = results

	event := mb.Event{}
	event.RootFields = common.MapStr{
		"cluster_uuid":   info.ClusterID,
		"timestamp":      common.Time(time.Now()),
		"interval_ms":    m.Module().Config().Period / time.Millisecond,
		"type":           "index_recovery",
		"index_recovery": indexRecovery,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
	r.Event(event)

	return nil
}
