package status

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	// Schema for mapping all Galera-Status-Variables
	schema_full = s.Schema{
		"apply": s.Object{
			"oooe": c.Float("wsrep_apply_oooe"),
			"oool": c.Float("wsrep_apply_oool"),
			"window": c.Float("wsrep_apply_window"),
		},
		"cert": s.Object{
			"deps_distance": c.Float("wsrep_cert_deps_distance"),
			"index_size": c.Int("wsrep_cert_index_size"),
			"interval": c.Float("wsrep_cert_interval"),
		},
		"cluster": s.Object{
			"conf_id": c.Int("wsrep_cluster_conf_id"),
			"size": c.Int("wsrep_cluster_size"),
			"status": c.Str("wsrep_cluster_status"),
		},
		"commit": s.Object{
			"oooe": c.Float("wsrep_commit_oooe"),
			"window": c.Float("wsrep_commit_window"),
		},
		"connected": c.Str("wsrep_connected"),
		"evs": s.Object{
			"evict": c.Str("wsrep_evs_evict_list"),
			"state": c.Str("wsrep_evs_state"),
		},
		"flow_ctl": s.Object{
			"paused": c.Float("wsrep_flow_control_paused"),
			"paused_ns": c.Int("wsrep_flow_control_paused_ns"),
			"recv": c.Int("wsrep_flow_control_recv"),
			"sent": c.Int("wsrep_flow_control_sent"),
		},
		"last_committed": c.Int("wsrep_last_committed"),
		"local": s.Object{
			"bf_aborts": c.Int("wsrep_local_bf_aborts"),
			"cert_failures": c.Int("wsrep_local_cert_failures"),
			"commits": c.Int("wsrep_local_commits"),
			"recv": s.Object{
				"queue":     c.Int("wsrep_local_recv_queue"),
				"queue_avg": c.Float("wsrep_local_recv_queue_avg"),
				"queue_max": c.Int("wsrep_local_recv_queue_max"),
				"queue_min": c.Int("wsrep_local_recv_queue_min"),
			},
			"replays": c.Int("wsrep_local_replays"),
			"send": s.Object{
				"queue": c.Int("wsrep_local_send_queue"),
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
			"data_bytes": c.Int("wsrep_repl_data_bytes"),
			"keys": c.Int("wsrep_repl_keys"),
			"keys_bytes": c.Int("wsrep_repl_keys_bytes"),
			"other_bytes": c.Int("wsrep_repl_other_bytes"),
			"count": c.Int("wsrep_replicated"),
			"bytes": c.Int("wsrep_replicated_bytes"),
		},
	}

	// Schema for mapping Galera-Status-Variables related to cluster health
	schema_small = s.Schema{
		"local" : s.Object{
			"state" : c.Str("wsrep_local_state_comment"),
		},
		"evs": s.Object{
			"state": c.Str("wsrep_evs_state"),
		},
		"cluster": s.Object{
			"size": c.Int("wsrep_cluster_size"),
			"status": c.Str("wsrep_cluster_status"),
		},
		"connected": c.Str("wsrep_connected"),
		"ready": c.Str("wsrep_ready"),
	}
)

// Map data to MapStr of server stats variables: http://galeracluster.com/documentation-webpages/galerastatusvariables.html
// queryMode specifies, which subset of the available Variables is used.
func eventMapping(status map[string]string, queryMode string) (common.MapStr, error) {
	source := map[string]interface{}{}
	for key, val := range status {
		source[key] = val
	}

	schema, err := getSchemaforMode(queryMode)
	if err != nil {
		return nil, err
	}

	data, _ := schema.Apply(source)
	return data, nil
}

// Maps all variables from the status fetch which are not in the predefined schema
func rawEventMapping(status map[string]string, queryMode string) (common.MapStr, error) {
	source := common.MapStr{}

	schema, err := getSchemaforMode(queryMode)
	if err != nil {
		return nil, err
	}

	for key, val := range status {
		// Only adds events which are not in the mapping
		if schema.HasKey(key) {
			continue
		}

		source[key] = val
	}

	return source, nil
}

// Returns the appropriate schema map for the query method
func getSchemaforMode(queryMode string) (s.Schema, error) {
	switch queryMode {
	case "full":
		return schema_full, nil
	case "small":
		return schema_small, nil
	default:
		return nil, fmt.Errorf("Illegal query mode: %s", queryMode)
	}
}
