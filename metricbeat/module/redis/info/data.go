package info

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {

	// Full mapping from info
	event := common.MapStr{
		"clients": common.MapStr{
			"connected":           h.ToInt("connected_clients", info),
			"longest_output_list": h.ToInt("client_longest_output_list", info),
			"biggest_input_buf":   h.ToInt("client_biggest_input_buf", info),
			"blocked":             h.ToInt("blocked_clients", info),
		},
		"cluster": common.MapStr{
			"enabled": h.ToBool("cluster_enabled", info),
		},
		"cpu": common.MapStr{
			"used": common.MapStr{
				"sys":           h.ToFloat("used_cpu_sys", info),
				"user":          h.ToFloat("used_cpu_user", info),
				"sys_children":  h.ToFloat("used_cpu_sys_children", info),
				"user_children": h.ToFloat("used_cpu_user_children", info),
			},
		},
		"memory": common.MapStr{
			"used": common.MapStr{
				"value": h.ToInt("used_memory", info), // As it is a top key, this goes into value
				"rss":   h.ToInt("used_memory_rss", info),
				"peak":  h.ToInt("used_memory_peak", info),
				"lua":   h.ToInt("used_memory_lua", info),
			},
			"allocator": h.ToStr("mem_allocator", info), // Could be moved to server as it rarely changes
		},
		"persistence": common.MapStr{
			"loading": h.ToBool("loading", info),
			"rdb": common.MapStr{
				"changes_since_last_save": h.ToInt("rdb_changes_since_last_save", info),
				"bgsave_in_progress":      h.ToBool("rdb_bgsave_in_progress", info),
				"last_save_time":          h.ToInt("rdb_last_save_time", info),
				"last_bgsave_status":      h.ToStr("rdb_last_bgsave_status", info),
				"last_bgsave_time_sec":    h.ToInt("rdb_last_bgsave_time_sec", info),
				"current_bgsave_time_sec": h.ToInt("rdb_current_bgsave_time_sec", info),
			},
			"used": common.MapStr{
				"enabled":                  h.ToBool("aof_enabled", info),
				"rewrite_in_progress":      h.ToBool("aof_rewrite_in_progress", info),
				"rewrite_scheduled":        h.ToBool("aof_rewrite_scheduled", info),
				"last_rewrite_time_sec":    h.ToInt("aof_last_rewrite_time_sec", info),
				"current_rewrite_time_sec": h.ToInt("aof_current_rewrite_time_sec", info),
				"last_bgrewrite_status":    h.ToStr("aof_last_bgrewrite_status", info),
				"last_write_status":        h.ToStr("aof_last_write_status", info),
			},
		},
		"replication": common.MapStr{
			"role":             h.ToStr("role", info),
			"connected_slaves": h.ToInt("connected_slaves", info),
			"master_offset":    h.ToInt("master_repl_offset", info),
			"backlog": common.MapStr{
				"active":            h.ToInt("repl_backlog_active", info),
				"size":              h.ToInt("repl_backlog_size", info),
				"first_byte_offset": h.ToInt("repl_backlog_first_byte_offset", info),
				"histlen":           h.ToInt("repl_backlog_histlen", info),
			},
		},
		"server": common.MapStr{
			"version":          h.ToStr("redis_version", info),
			"git_sha1":         h.ToStr("redis_git_sha1", info),
			"git_dirty":        h.ToStr("redis_git_dirty", info),
			"build_id":         h.ToStr("redis_build_id", info),
			"mode":             h.ToStr("redis_mode", info),
			"os":               h.ToStr("os", info),
			"arch_bits":        h.ToStr("arch_bits", info),
			"multiplexing_api": h.ToStr("multiplexing_api", info),
			"gcc_version":      h.ToStr("gcc_version", info),
			"process_id":       h.ToInt("process_id", info),
			"run_id":           h.ToStr("run_id", info),
			"tcp_port":         h.ToInt("tcp_port", info),
			"uptime":           h.ToInt("uptime_in_seconds", info), // Uptime days was removed as duplicate
			"hz":               h.ToInt("hz", info),
			"lru_clock":        h.ToInt("lru_clock", info),
			"config_file":      h.ToStr("config_file", info),
		},
		"stats": common.MapStr{
			"connections": common.MapStr{
				"received": h.ToInt("total_connections_received", info),
				"rejected": h.ToInt("rejected_connections", info),
			},
			"total_commands_processed":  h.ToInt("total_commands_processed", info),
			"total_net_input_bytes":     h.ToInt("total_net_input_bytes", info),
			"total_net_output_bytes":    h.ToInt("total_net_output_bytes", info),
			"instantaneous_ops_per_sec": h.ToInt("instantaneous_ops_per_sec", info),
			"instantaneous_input_kbps":  h.ToFloat("instantaneous_input_kbps", info),
			"instantaneous_output_kbps": h.ToFloat("instantaneous_output_kbps", info),
			"sync": common.MapStr{
				"full":        h.ToInt("sync_full", info),
				"partial_ok":  h.ToInt("sync_partial_ok", info),
				"partial_err": h.ToInt("sync_partial_err", info),
			},
			"keys": common.MapStr{
				"expired": h.ToInt("expired_keys", info),
				"evicted": h.ToInt("evicted_keys", info),
			},
			"keyspace": common.MapStr{
				"hits":   h.ToInt("keyspace_hits", info),
				"misses": h.ToInt("keyspace_misses", info),
			},
			"pubsub_channels":        h.ToInt("pubsub_channels", info),
			"pubsub_patterns":        h.ToInt("pubsub_patterns", info),
			"latest_fork_usec":       h.ToInt("latest_fork_usec", info),
			"migrate_cached_sockets": h.ToInt("migrate_cached_sockets", info),
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
				"keys":    h.ToInt("keys", db),
				"expires": h.ToInt("expires", db),
				"avg_ttl": h.ToInt("avg_ttl", db),
			}
		}
	}
	return keyspace
}
