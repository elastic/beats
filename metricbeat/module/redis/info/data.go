package info

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"clients": s.Object{
			"connected":           c.Int("connected_clients"),
			"longest_output_list": c.Int("client_longest_output_list"),
			"biggest_input_buf":   c.Int("client_biggest_input_buf"),
			"blocked":             c.Int("blocked_clients"),
		},
		"cluster": s.Object{
			"enabled": c.Bool("cluster_enabled"),
		},
		"cpu": s.Object{
			"used": s.Object{
				"sys":           c.Float("used_cpu_sys"),
				"user":          c.Float("used_cpu_user"),
				"sys_children":  c.Float("used_cpu_sys_children"),
				"user_children": c.Float("used_cpu_user_children"),
			},
		},
		"memory": s.Object{
			"used": s.Object{
				"value": c.Int("used_memory"), // As it is a top key, this goes into value
				"rss":   c.Int("used_memory_rss"),
				"peak":  c.Int("used_memory_peak"),
				"lua":   c.Int("used_memory_lua"),
			},
			"allocator": c.Str("mem_allocator"), // Could be moved to server as it rarely changes
		},
		"persistence": s.Object{
			"loading": c.Bool("loading"),
			"rdb": s.Object{
				"last_save.changes_since": c.Int("rdb_changes_since_last_save"),
				"last_save.time":          c.Int("rdb_last_save_time"),
				"bgsave": s.Object{
					"last_status":      c.Str("rdb_last_bgsave_status"),
					"in_progress":      c.Bool("rdb_bgsave_in_progress"),
					"last_time.sec":    c.Int("rdb_last_bgsave_time_sec"),
					"current_time.sec": c.Int("rdb_current_bgsave_time_sec"),
				},
			},
			"aof": s.Object{
				"enabled": c.Bool("aof_enabled"),
				"rewrite": s.Object{
					"in_progress":      c.Bool("aof_rewrite_in_progress"),
					"scheduled":        c.Bool("aof_rewrite_scheduled"),
					"last_time.sec":    c.Int("aof_last_rewrite_time_sec"),
					"current_time.sec": c.Int("aof_current_rewrite_time_sec"),
				},
				"bgrewrite.last_status": c.Str("aof_last_bgrewrite_status"),
				"write.last_status":     c.Str("aof_last_write_status"),
			},
		},
		"replication": s.Object{
			"role":             c.Str("role"),
			"connected_slaves": c.Int("connected_slaves"),
			"master_offset":    c.Int("master_repl_offset"),
			"backlog": s.Object{
				"active":            c.Int("repl_backlog_active"),
				"size":              c.Int("repl_backlog_size"),
				"first_byte_offset": c.Int("repl_backlog_first_byte_offset"),
				"histlen":           c.Int("repl_backlog_histlen"),
			},
		},
		"server": s.Object{
			"version":          c.Str("redis_version"),
			"git_sha1":         c.Str("redis_git_sha1"),
			"git_dirty":        c.Str("redis_git_dirty"),
			"build_id":         c.Str("redis_build_id"),
			"mode":             c.Str("redis_mode"),
			"os":               c.Str("os"),
			"arch_bits":        c.Str("arch_bits"),
			"multiplexing_api": c.Str("multiplexing_api"),
			"gcc_version":      c.Str("gcc_version"),
			"process_id":       c.Int("process_id"),
			"run_id":           c.Str("run_id"),
			"tcp_port":         c.Int("tcp_port"),
			"uptime":           c.Int("uptime_in_seconds"), // Uptime days was removed as duplicate
			"hz":               c.Int("hz"),
			"lru_clock":        c.Int("lru_clock"),
			"config_file":      c.Str("config_file"),
		},
		"stats": s.Object{
			"connections": s.Object{
				"received": c.Int("total_connections_received"),
				"rejected": c.Int("rejected_connections"),
			},
			"commands_processed": c.Int("total_commands_processed"),
			"net.input.bytes":    c.Int("total_net_input_bytes"),
			"net.output.bytes":   c.Int("total_net_output_bytes"),
			"instantaneous": s.Object{
				"ops_per_sec": c.Int("instantaneous_ops_per_sec"),
				"input_kbps":  c.Float("instantaneous_input_kbps"),
				"output_kbps": c.Float("instantaneous_output_kbps"),
			},
			"sync": s.Object{
				"full":        c.Int("sync_full"),
				"partial.ok":  c.Int("sync_partial_ok"),
				"partial.err": c.Int("sync_partial_err"),
			},
			"keys": s.Object{
				"expired": c.Int("expired_keys"),
				"evicted": c.Int("evicted_keys"),
			},
			"keyspace": s.Object{
				"hits":   c.Int("keyspace_hits"),
				"misses": c.Int("keyspace_misses"),
			},
			"pubsub.channels":        c.Int("pubsub_channels"),
			"pubsub.patterns":        c.Int("pubsub_patterns"),
			"latest_fork_usec":       c.Int("latest_fork_usec"),
			"migrate_cached_sockets": c.Int("migrate_cached_sockets"),
		},
	}
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {
	// Full mapping from info
	source := map[string]interface{}{}
	for key, val := range info {
		source[key] = val
	}
	return schema.Apply(source)
}
