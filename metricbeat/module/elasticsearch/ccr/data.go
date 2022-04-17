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

	"github.com/menderesk/beats/v7/metricbeat/helper/elastic"

	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"leader": s.Object{
			"index":             c.Str("leader_index"),
			"max_seq_no":        c.Int("leader_max_seq_no"),
			"global_checkpoint": c.Int("leader_global_checkpoint"),
		},
		"total_time": s.Object{
			"read": s.Object{
				"ms": c.Int("total_read_time_millis"),
				"remote_exec": s.Object{
					"ms": c.Int("total_read_remote_exec_time_millis"),
				},
			},

			"write": s.Object{
				"ms": c.Int("total_write_time_millis"),
			},
		},
		"write_buffer": s.Object{
			"size": s.Object{
				"bytes": c.Int("write_buffer_size_in_bytes"),
			},
			"operation": s.Object{
				"count": c.Int("write_buffer_operation_count"),
			},
		},
		"bytes_read": c.Int("bytes_read"),
		"follower": s.Object{
			"index": c.Str("follower_index"),
			"shard": s.Object{
				"number": c.Int("shard_id"),
			},
			"operations_written": c.Int("operations_written"),
			"operations": s.Object{
				"read": s.Object{
					"count": c.Int("operations_read"),
				},
			},
			"max_seq_no": c.Int("follower_max_seq_no"),
			"time_since_last_read": s.Object{
				"ms": c.Int("time_since_last_read_millis"),
			},
			"global_checkpoint": c.Int("follower_global_checkpoint"),
			"settings_version":  c.Int("follower_settings_version"),
			"aliases_version":   c.Int("follower_aliases_version"),
		},
		"read_exceptions": c.Ifc("read_exceptions"),
		"requests": s.Object{
			"successful": s.Object{
				"read": s.Object{
					"count": c.Int("successful_read_requests"),
				},
				"write": s.Object{
					"count": c.Int("successful_write_requests"),
				},
			},
			"failed": s.Object{
				"read": s.Object{
					"count": c.Int("failed_read_requests"),
				},
				"write": s.Object{
					"count": c.Int("failed_write_requests"),
				},
			},
			"outstanding": s.Object{
				"read": s.Object{
					"count": c.Int("outstanding_read_requests"),
				},
				"write": s.Object{
					"count": c.Int("outstanding_write_requests"),
				},
			},
		},
	}

	autoFollowSchema = s.Schema{
		"failed": s.Object{
			"follow_indices": s.Object{
				"count": c.Int("number_of_failed_follow_indices"),
			},
			"remote_cluster_state_requests": s.Object{
				"count": c.Int("number_of_failed_remote_cluster_state_requests"),
			},
		},
		"success": s.Object{
			"follow_indices": s.Object{
				"count": c.Int("number_of_successful_follow_indices"),
			},
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

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool) error {
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
			event.ModuleFields = common.MapStr{}

			event.RootFields.Put("service.name", elasticsearch.ModuleName)
			event.ModuleFields.Put("cluster.name", info.ClusterName)
			event.ModuleFields.Put("cluster.id", info.ClusterID)

			event.MetricSetFields, _ = schema.Apply(followerShard)

			autoFollow, _ := autoFollowSchema.Apply(data.AutoFollowStats)
			event.MetricSetFields["auto_follow"] = autoFollow

			// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
			// When using Agent, the index name is overwritten anyways.
			if isXpack {
				index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
				event.Index = index
			}

			r.Event(event)
		}
	}

	return errs.Err()
}
