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

package galera_status

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	// Schema for mapping all Galera-Status-Variables
	schema = s.Schema{
		"apply": s.Object{
			"oooe":   c.Float("wsrep_apply_oooe"),
			"oool":   c.Float("wsrep_apply_oool"),
			"window": c.Float("wsrep_apply_window"),
		},
		"cert": s.Object{
			"deps_distance": c.Float("wsrep_cert_deps_distance"),
			"index_size":    c.Int("wsrep_cert_index_size"),
			"interval":      c.Float("wsrep_cert_interval"),
		},
		"cluster": s.Object{
			"conf_id": c.Int("wsrep_cluster_conf_id"),
			"size":    c.Int("wsrep_cluster_size"),
			"status":  c.Str("wsrep_cluster_status"),
		},
		"commit": s.Object{
			"oooe":   c.Float("wsrep_commit_oooe"),
			"window": c.Float("wsrep_commit_window"),
		},
		"connected": c.Str("wsrep_connected"),
		"evs": s.Object{
			"evict": c.Str("wsrep_evs_evict_list"),
			"state": c.Str("wsrep_evs_state"),
		},
		"flow_ctl": s.Object{
			"paused":    c.Float("wsrep_flow_control_paused"),
			"paused_ns": c.Int("wsrep_flow_control_paused_ns"),
			"recv":      c.Int("wsrep_flow_control_recv"),
			"sent":      c.Int("wsrep_flow_control_sent"),
		},
		"last_committed": c.Int("wsrep_last_committed"),
		"local": s.Object{
			"bf_aborts":     c.Int("wsrep_local_bf_aborts"),
			"cert_failures": c.Int("wsrep_local_cert_failures"),
			"commits":       c.Int("wsrep_local_commits"),
			"recv": s.Object{
				"queue":     c.Int("wsrep_local_recv_queue"),
				"queue_avg": c.Float("wsrep_local_recv_queue_avg"),
				"queue_max": c.Int("wsrep_local_recv_queue_max"),
				"queue_min": c.Int("wsrep_local_recv_queue_min"),
			},
			"replays": c.Int("wsrep_local_replays"),
			"send": s.Object{
				"queue":     c.Int("wsrep_local_send_queue"),
				"queue_avg": c.Float("wsrep_local_send_queue_avg"),
				"queue_max": c.Int("wsrep_local_send_queue_max"),
				"queue_min": c.Int("wsrep_local_send_queue_min"),
			},
			"state": c.Str("wsrep_local_state_comment"),
		},
		"ready": c.Str("wsrep_ready"),
		"received": s.Object{
			"count": c.Int("wsrep_received"),
			"bytes": c.Int("wsrep_received_bytes"),
		},
		"repl": s.Object{
			"data_bytes":  c.Int("wsrep_repl_data_bytes"),
			"keys":        c.Int("wsrep_repl_keys"),
			"keys_bytes":  c.Int("wsrep_repl_keys_bytes"),
			"other_bytes": c.Int("wsrep_repl_other_bytes"),
			"count":       c.Int("wsrep_replicated"),
			"bytes":       c.Int("wsrep_replicated_bytes"),
		},
	}
)

// Map data to MapStr of server stats variables: http://galeracluster.com/documentation-webpages/galerastatusvariables.html
// queryMode specifies, which subset of the available Variables is used.
func eventMapping(status map[string]string) common.MapStr {
	source := map[string]interface{}{}
	for key, val := range status {
		source[key] = val
	}

	data, _ := schema.Apply(source)
	return data
}

// Maps all variables from the status fetch which are not in the predefined schema
func rawEventMapping(status map[string]string) common.MapStr {
	source := common.MapStr{}
	for key, val := range status {
		// Only adds events which are not in the mapping
		if schema.HasKey(key) {
			continue
		}

		source[key] = val
	}

	return source
}
