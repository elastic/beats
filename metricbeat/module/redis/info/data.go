package info

import (
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {

	// Full mapping from info
	event := common.MapStr{
		"clients": common.MapStr{
			"connected_clients":          toInt(info["connected_clients"]),
			"client_longest_output_list": toInt(info["client_longest_output_list"]),
			"client_biggest_input_buf":   toInt(info["client_biggest_input_buf"]),
			"blocked_clients":            toInt(info["blocked_clients"]),
		},
		"cluster": common.MapStr{
			"cluster_enabled": toBool(info["cluster_enabled"]),
		},
		"cpu": common.MapStr{
			"used_cpu_sys":           toFloat(info["used_cpu_sys"]),
			"used_cpu_user":          toFloat(info["used_cpu_user"]),
			"used_cpu_sys_children":  toFloat(info["used_cpu_sys_children"]),
			"used_cpu_user_children": toFloat(info["used_cpu_user_children"]),
		},
		"memory": common.MapStr{
			"used_memory":      toInt(info["used_memory"]),
			"used_memory_rss":  toInt(info["used_memory_rss"]),
			"used_memory_peak": toInt(info["used_memory_peak"]),
			"used_memory_lua":  toInt(info["used_memory_lua"]),
			"mem_allocator":    info["mem_allocator"], // Could be moved to server as it rarely changes
		},
		"persistence": common.MapStr{
			"loading":                      toBool(info["loading"]),
			"rdb_changes_since_last_save":  toInt(info["rdb_changes_since_last_save"]),
			"rdb_bgsave_in_progress":       toBool(info["rdb_bgsave_in_progress"]),
			"rdb_last_save_time":           toInt(info["rdb_last_save_time"]),
			"rdb_last_bgsave_status":       info["rdb_last_bgsave_status"],
			"rdb_last_bgsave_time_sec":     toInt(info["rdb_last_bgsave_time_sec"]),
			"rdb_current_bgsave_time_sec":  toInt(info["rdb_current_bgsave_time_sec"]),
			"aof_enabled":                  toBool(info["aof_enabled"]),
			"aof_rewrite_in_progress":      toBool(info["aof_rewrite_in_progress"]),
			"aof_rewrite_scheduled":        toBool(info["aof_rewrite_scheduled"]),
			"aof_last_rewrite_time_sec":    toInt(info["aof_last_rewrite_time_sec"]),
			"aof_current_rewrite_time_sec": toInt(info["aof_current_rewrite_time_sec"]),
			"aof_last_bgrewrite_status":    info["aof_last_bgrewrite_status"],
			"aof_last_write_status":        info["aof_last_write_status"],
		},
		"replication": common.MapStr{
			"role":                           info["role"],
			"connected_slaves":               toInt(info["connected_slaves"]),
			"master_repl_offset":             toInt(info["master_repl_offset"]),
			"repl_backlog_active":            toInt(info["repl_backlog_active"]),
			"repl_backlog_size":              toInt(info["repl_backlog_size"]),
			"repl_backlog_first_byte_offset": toInt(info["repl_backlog_first_byte_offset"]),
			"repl_backlog_histlen":           toInt(info["repl_backlog_histlen"]),
		},
		"server": common.MapStr{
			"redis_version":     info["redis_version"],
			"redis_git_sha1":    info["redis_git_sha1"],
			"redis_git_dirty":   info["redis_git_dirty"],
			"redis_build_id":    info["redis_build_id"],
			"redis_mode":        info["redis_mode"],
			"os":                info["os"],
			"arch_bits":         info["arch_bits"],
			"multiplexing_api":  info["multiplexing_api"],
			"gcc_version":       info["gcc_version"],
			"process_id":        toInt(info["process_id"]),
			"run_id":            info["run_id"],
			"tcp_port":          toInt(info["tcp_port"]),
			"uptime_in_seconds": toInt(info["uptime_in_seconds"]),
			"uptime_in_days":    toInt(info["uptime_in_days"]),
			"hz":                toInt(info["hz"]),
			"lru_clock":         toInt(info["lru_clock"]),
			"config_file":       info["config_file"],
		},
		"stats": common.MapStr{
			"total_connections_received": toInt(info["total_connections_received"]),
			"total_commands_processed":   toInt(info["total_commands_processed"]),
			"instantaneous_ops_per_sec":  toInt(info["instantaneous_ops_per_sec"]),
			"total_net_input_bytes":      toInt(info["total_net_input_bytes"]),
			"total_net_output_bytes":     toInt(info["total_net_output_bytes"]),
			"instantaneous_input_kbps":   toFloat(info["instantaneous_input_kbps"]),
			"instantaneous_output_kbps":  toFloat(info["instantaneous_output_kbps"]),
			"rejected_connections":       toInt(info["rejected_connections"]),
			"sync_full":                  toInt(info["sync_full"]),
			"sync_partial_ok":            toInt(info["sync_partial_ok"]),
			"sync_partial_err":           toInt(info["sync_partial_err"]),
			"expired_keys":               toInt(info["expired_keys"]),
			"evicted_keys":               toInt(info["evicted_keys"]),
			"keyspace_hits":              toInt(info["keyspace_hits"]),
			"keyspace_misses":            toInt(info["keyspace_misses"]),
			"pubsub_channels":            toInt(info["pubsub_channels"]),
			"pubsub_patterns":            toInt(info["pubsub_patterns"]),
			"latest_fork_usec":           toInt(info["latest_fork_usec"]),
			"migrate_cached_sockets":     toInt(info["migrate_cached_sockets"]),
		},
		"keyspace": getKeyspaceStats(info),
	}

	return event
}

func getKeyspaceStats(info map[string]string) map[string]common.MapStr {
	keyspaceMap := findKeyspaceStats(info)
	return parseKeyspaceStats(keyspaceMap)
}

// findKeyspaceStats will grep for keyspace ("^db" keys) and return the resulting map
func findKeyspaceStats(info map[string]string) map[string]string {
	keyspace := map[string]string{}

	for k, v := range info {
		if strings.HasPrefix(k, "db") {
			keyspace[k] = v
		}
	}
	return keyspace
}

// parseKeyspaceStats resolves the overloaded value string that Redis returns for keyspace
func parseKeyspaceStats(keyspaceMap map[string]string) map[string]common.MapStr {
	keyspace := map[string]common.MapStr{}
	for k, v := range keyspaceMap {

		// Extract out the overloaded values for db keyspace
		// fmt: info[db0] = keys=795341,expires=0,avg_ttl=0
		dbInfo := parseRedisLine(v, ",")

		if len(dbInfo) == 3 {
			db := map[string]string{}
			for _, dbEntry := range dbInfo {
				stats := parseRedisLine(dbEntry, "=")

				if len(stats) == 2 {
					db[stats[0]] = stats[1]
				}
			}
			keyspace[k] = common.MapStr{
				"keys":    toInt(db["keys"]),
				"expires": toInt(db["expires"]),
				"avg_ttl": toInt(db["avg_ttl"]),
			}
		}
	}
	return keyspace
}

// toInt converts value to int. In case of error, returns 0
func toInt(param string) int {
	value, err := strconv.Atoi(param)

	if err != nil {
		logp.Err("Error converting param to int: %s", param)
		value = 0
	}

	return value
}

// toBool converts value to bool. In case of error, returns false
func toBool(param string) bool {
	value, err := strconv.ParseBool(param)

	if err != nil {
		logp.Err("Error converting param to bool: %s", param)
		value = false
	}

	return value
}

// toFloat converts value to float64. In case of error, returns 0.0
func toFloat(param string) float64 {
	value, err := strconv.ParseFloat(param, 64)

	if err != nil {
		logp.Err("Error converting param to float: %s", param)
		value = 0.0
	}

	return value
}
