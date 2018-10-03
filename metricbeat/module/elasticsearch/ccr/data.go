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

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"leader": s.Object{
			"index":      c.Str("leader_index"),
			"max_seq_no": c.Int("leader_max_seq_no"),
		},
		"follower": s.Object{
			"index": c.Str("follower_index"),
			"shard": s.Object{
				"number": c.Int("shard_id"),
			},
			"operations_indexed": c.Int("number_of_operations_indexed"),
			"time_since_last_fetch": s.Object{
				"ms": c.Int("time_since_last_fetch_millis"),
			},
			"global_checkpoint": c.Int("follower_global_checkpoint"),
		},
	}
)

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
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
			event.MetricSetFields, err = schema.Apply(shard)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			event.RootFields = common.MapStr{}
			event.RootFields.Put("service.name", "elasticsearch")

			event.ModuleFields = common.MapStr{}
			event.ModuleFields.Put("cluster.name", info.ClusterName)
			event.ModuleFields.Put("cluster.id", info.ClusterID)

			r.Event(event)
		}
	}

	return errors.Err()
}
