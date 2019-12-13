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

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

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
			"operations_written": c.Int("operations_written"),
			"time_since_last_read": s.Object{
				"ms": c.Int("time_since_last_read_millis"),
			},
			"global_checkpoint": c.Int("follower_global_checkpoint"),
		},
	}
)

type response struct {
	AutoFollowStats map[string]interface{} `json:"auto_follow_stats"`
	FollowStats     struct {
		Indices []struct {
			Shards []map[string]interface{} `json:"shards"`
		} `json:"indices"`
	} `json:"follow_stats"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch CCR Stats API response")
	}

	var errs multierror.Errors
	for _, followerIndex := range data.FollowStats.Indices {
		for _, followerShard := range followerIndex.Shards {
			event := mb.Event{}
			event.RootFields = common.MapStr{}
			event.RootFields.Put("service.name", elasticsearch.ModuleName)

			event.ModuleFields = common.MapStr{}
			event.ModuleFields.Put("cluster.name", info.ClusterName)
			event.ModuleFields.Put("cluster.id", info.ClusterID)

			event.MetricSetFields, err = schema.Apply(followerShard)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "failure applying shard schema"))
				continue
			}

			r.Event(event)
		}
	}

	return errs.Err()
}
