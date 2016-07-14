package info

import (
	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

var (
	schema = h.NewSchema(common.MapStr{
		"clients": common.MapStr{
			"connected":           h.Int("connected_clients"),
			"longest_output_list": h.Int("client_longest_output_list"),
			"biggest_input_buf":   h.Int("client_biggest_input_buf"),
			"blocked":             h.Int("blocked_clients"),
		},
		"cluster": common.MapStr{
			"enabled": h.Bool("cluster_enabled"),
		},
		"cpu": common.MapStr{
			"used": common.MapStr{
				"sys":           h.Float("used_cpu_sys"),
				"user":          h.Float("used_cpu_user"),
				"sys_children":  h.Float("used_cpu_sys_children"),
				"user_children": h.Float("used_cpu_user_children"),
			},
		},
		"memory": common.MapStr{
			"used": common.MapStr{
				"value": h.Int("used_memory"), // As it is a top key, this goes into value
				"rss":   h.Int("used_memory_rss"),
				"peak":  h.Int("used_memory_peak"),
				"lua":   h.Int("used_memory_lua"),
			},
			"allocator": h.Str("mem_allocator"), // Could be moved to server as it rarely changes
		},
		"persistence": common.MapStr{
			"loading": h.Bool("loading"),
			"rdb": common.MapStr{
				"changes_since_last_save": h.Int("rdb_changes_since_last_save"),
				"bgsave_in_progress":      h.Bool("rdb_bgsave_in_progress"),
				"last_save_time":          h.Int("rdb_last_save_time"),
				"last_bgsave_status":      h.Str("rdb_last_bgsave_status"),
				"last_bgsave_time_sec":    h.Int("rdb_last_bgsave_time_sec"),
				"current_bgsave_time_sec": h.Int("rdb_current_bgsave_time_sec"),
			},
			"used": common.MapStr{
				"enabled":                  h.Bool("aof_enabled"),
				"rewrite_in_progress":      h.Bool("aof_rewrite_in_progress"),
				"rewrite_scheduled":        h.Bool("aof_rewrite_scheduled"),
				"last_rewrite_time_sec":    h.Int("aof_last_rewrite_time_sec"),
				"current_rewrite_time_sec": h.Int("aof_current_rewrite_time_sec"),
				"last_bgrewrite_status":    h.Str("aof_last_bgrewrite_status"),
				"last_write_status":        h.Str("aof_last_write_status"),
			},
		},
		"replication": common.MapStr{
			"role":             h.Str("role"),
			"connected_slaves": h.Int("connected_slaves"),
			"master_offset":    h.Int("master_repl_offset"),
			"backlog": common.MapStr{
				"active":            h.Int("repl_backlog_active"),
				"size":              h.Int("repl_backlog_size"),
				"first_byte_offset": h.Int("repl_backlog_first_byte_offset"),
				"histlen":           h.Int("repl_backlog_histlen"),
			},
		},
		"server": common.MapStr{
			"version":          h.Str("redis_version"),
			"git_sha1":         h.Str("redis_git_sha1"),
			"git_dirty":        h.Str("redis_git_dirty"),
			"build_id":         h.Str("redis_build_id"),
			"mode":             h.Str("redis_mode"),
			"os":               h.Str("os"),
			"arch_bits":        h.Str("arch_bits"),
			"multiplexing_api": h.Str("multiplexing_api"),
			"gcc_version":      h.Str("gcc_version"),
			"process_id":       h.Int("process_id"),
			"run_id":           h.Str("run_id"),
			"tcp_port":         h.Int("tcp_port"),
			"uptime":           h.Int("uptime_in_seconds"), // Uptime days was removed as duplicate
			"hz":               h.Int("hz"),
			"lru_clock":        h.Int("lru_clock"),
			"config_file":      h.Str("config_file"),
		},
		"stats": common.MapStr{
			"connections": common.MapStr{
				"received": h.Int("total_connections_received"),
				"rejected": h.Int("rejected_connections"),
			},
			"total_commands_processed":  h.Int("total_commands_processed"),
			"total_net_input_bytes":     h.Int("total_net_input_bytes"),
			"total_net_output_bytes":    h.Int("total_net_output_bytes"),
			"instantaneous_ops_per_sec": h.Int("instantaneous_ops_per_sec"),
			"instantaneous_input_kbps":  h.Float("instantaneous_input_kbps"),
			"instantaneous_output_kbps": h.Float("instantaneous_output_kbps"),
			"sync": common.MapStr{
				"full":        h.Int("sync_full"),
				"partial_ok":  h.Int("sync_partial_ok"),
				"partial_err": h.Int("sync_partial_err"),
			},
			"keys": common.MapStr{
				"expired": h.Int("expired_keys"),
				"evicted": h.Int("evicted_keys"),
			},
			"keyspace": common.MapStr{
				"hits":   h.Int("keyspace_hits"),
				"misses": h.Int("keyspace_misses"),
			},
			"pubsub_channels":        h.Int("pubsub_channels"),
			"pubsub_patterns":        h.Int("pubsub_patterns"),
			"latest_fork_usec":       h.Int("latest_fork_usec"),
			"migrate_cached_sockets": h.Int("migrate_cached_sockets"),
		},
	})
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {

	// Full mapping from info
	return schema.Apply(info)
}
