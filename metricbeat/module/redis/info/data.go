package info

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {

	// Full mapping from info
	event := common.MapStr{
		"clients": common.MapStr{
			"connected_clients":          toInt(info["connected_clients"]),
			"client_longest_output_list": info["client_longest_output_list"],
			"client_biggest_input_buf":   info["client_biggest_input_buf"],
			"blocked_clients":            info["blocked_clients"],
		},
		"cluster": common.MapStr{
			"cluster_enabled": info["cluster_enabled"],
		},
		"cpu": common.MapStr{
			"used_cpu_sys":           info["used_cpu_sys"],
			"used_cpu_user":          info["used_cpu_user"],
			"used_cpu_sys_children":  info["used_cpu_sys_children"],
			"used_cpu_user_children": info["used_cpu_user_children"],
		},
		"memory": common.MapStr{
			"used_memory":      toInt(info["used_memory"]),
			"used_memory_rss":  info["used_memory_rss"],
			"used_memory_peak": info["used_memory_peak"],
			"used_memory_lua":  info["used_memory_lua"],
			"mem_allocator":    info["mem_allocator"], // Could be moved server as it rarely changes
		},
		"presistence": common.MapStr{
			"loading":                      info["loading"],
			"rdb_changes_since_last_save":  info["rdb_changes_since_last_save"],
			"rdb_bgsave_in_progress":       info["rdb_bgsave_in_progress"],
			"rdb_last_save_time":           info["rdb_last_save_time"],
			"rdb_last_bgsave_status":       info["rdb_last_bgsave_status"],
			"rdb_last_bgsave_time_sec":     info["rdb_last_bgsave_time_sec"],
			"rdb_current_bgsave_time_sec":  info["rdb_current_bgsave_time_sec"],
			"aof_enabled":                  info["aof_enabled"],
			"aof_rewrite_in_progress":      info["aof_rewrite_in_progress"],
			"aof_rewrite_scheduled":        info["aof_rewrite_scheduled"],
			"aof_last_rewrite_time_sec":    info["aof_last_rewrite_time_sec"],
			"aof_current_rewrite_time_sec": info["aof_current_rewrite_time_sec"],
			"aof_last_bgrewrite_status":    info["aof_last_bgrewrite_status"],
			"aof_last_write_status":        info["aof_last_write_status"],
		},
		"replication": common.MapStr{
			"role":                           info["role"],
			"connected_slaves":               info["connected_slaves"],
			"master_repl_offset":             info["master_repl_offset"],
			"repl_backlog_active":            info["repl_backlog_active"],
			"repl_backlog_size":              info["repl_backlog_size"],
			"repl_backlog_first_byte_offset": info["repl_backlog_first_byte_offset"],
			"repl_backlog_histlen":           info["repl_backlog_histlen"],
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
			"process_id":        info["process_id"],
			"run_id":            info["run_id"],
			"tcp_port":          info["tcp_port"],
			"uptime_in_seconds": info["uptime_in_seconds"],
			"uptime_in_days":    info["uptime_in_days"],
			"hz":                info["hz"],
			"lru_clock":         info["lru_clock"],
			"config_file":       info["config_file"],
		},
		"stats": common.MapStr{
			"total_connections_received": toInt(info["total_connections_received"]),
			"total_commands_processed":   info["total_commands_processed"],
			"instantaneous_ops_per_sec":  info["instantaneous_ops_per_sec"],
			"total_net_input_bytes":      info["total_net_input_bytes"],
			"total_net_output_bytes":     info["total_net_output_bytes"],
			"instantaneous_input_kbps":   info["instantaneous_input_kbps"],
			"instantaneous_output_kbps":  info["instantaneous_output_kbps"],
			"rejected_connections":       toInt(info["rejected_connections"]),
			"sync_full":                  info["sync_full"],
			"sync_partial_ok":            info["sync_partial_ok"],
			"sync_partial_err":           info["sync_partial_err"],
			"expired_keys":               info["expired_keys"],
			"evicted_keys":               info["evicted_keys"],
			"keyspace_hits":              info["keyspace_hits"],

			"keyspace_misses":        info["keyspace_misses"],
			"pubsub_channels":        info["pubsub_channels"],
			"pubsub_patterns":        info["pubsub_patterns"],
			"latest_fork_usec":       info["latest_fork_usec"],
			"migrate_cached_sockets": info["migrate_cached_sockets"],
		},
	}

	return event
}

func toInt(param string) int {
	value, err := strconv.Atoi(param)

	if err != nil {
		logp.Err("Error converting param to int: %s", param)
		value = 0
	}

	return value
}
