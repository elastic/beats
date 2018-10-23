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
	"fmt"
	"time"

	"github.com/joeshaw/multierror"
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
		return errors.Wrap(err, "failure parsing Elasticsearch CCR Stats API response")
	}

	var errors multierror.Errors
	for _, followerShards := range data {

		shards, ok := followerShards.([]interface{})
		if !ok {
			err := fmt.Errorf("shards is not an array")
			errors = append(errors, err)
			continue
		}

		for _, s := range shards {
			shard, ok := s.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("shard is not an object")
				errors = append(errors, err)
				continue
			}
			event := mb.Event{}
			event.RootFields = common.MapStr{
				"cluster_uuid": info.ClusterID,
				"timestamp":    common.Time(time.Now()),
				"interval_ms":  m.Module().Config().Period / time.Millisecond,
				"type":         "ccr_stats",
				"ccr_stats":    shard,
			}

			event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
			r.Event(event)
		}
	}
	return errors.Err()
}
