package info

import (
	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

// Map data to MapStr
func eventMapping(info map[string]string) common.MapStr {

	// Full mapping from info
	errs := map[string]error{}
	event := common.MapStr{
		"clients": common.MapStr{
			"connected":           h.ToInt("connected_clients", info, errs, "clients.connected"),
			"longest_output_list": h.ToInt("client_longest_output_list", info, errs, "clients.longest_output_list"),
			"biggest_input_buf":   h.ToInt("client_biggest_input_buf", info, errs, "clients.biggest_input_buf"),
			"blocked":             h.ToInt("blocked_clients", info, errs, "clients.blocked"),
		},
		"cluster": common.MapStr{
			"enabled": h.ToBool("cluster_enabled", info, errs, "cluster.enabled"),
		},
		"cpu": common.MapStr{
			"used": common.MapStr{
				"sys":           h.ToFloat("used_cpu_sys", info, errs, "cpu.used.sys"),
				"user":          h.ToFloat("used_cpu_user", info, errs, "cpu.used.user"),
				"sys_children":  h.ToFloat("used_cpu_sys_children", info, errs, "cpu.used.sys_children"),
				"user_children": h.ToFloat("used_cpu_user_children", info, errs, "cpu.used.user_children"),
			},
		},
		"memory": common.MapStr{
			"used": common.MapStr{
				"value": h.ToInt("used_memory", info, errs, "memory.used.value"), // As it is a top key, this goes into value
				"rss":   h.ToInt("used_memory_rss", info, errs, "memory.used.rss"),
				"peak":  h.ToInt("used_memory_peak", info, errs, "memory.used.peak"),
				"lua":   h.ToInt("used_memory_lua", info, errs, "memory.used.lua"),
			},
			"allocator": h.ToStr("mem_allocator", info, errs, "memory.allocator"), // Could be moved to server as it rarely changes
		},
		"persistence": common.MapStr{
			"loading": h.ToBool("loading", info, errs, "persistence.loading"),
			"rdb": common.MapStr{
				"changes_since_last_save": h.ToInt("rdb_changes_since_last_save", info, errs, "persistence.rdp.changes_since_last_save"),
				"bgsave_in_progress":      h.ToBool("rdb_bgsave_in_progress", info, errs, "persistence.rdp.bgsave_in_progress"),
				"last_save_time":          h.ToInt("rdb_last_save_time", info, errs, "persistence.rdp.last_save_time"),
				"last_bgsave_status":      h.ToStr("rdb_last_bgsave_status", info, errs, "persistence.rdp.last_bgsave_status"),
				"last_bgsave_time_sec":    h.ToInt("rdb_last_bgsave_time_sec", info, errs, "persistence.rdp.last_bgsave_time_sec"),
				"current_bgsave_time_sec": h.ToInt("rdb_current_bgsave_time_sec", info, errs, "persistence.rdp.current_bgsave_time_sec"),
			},
			"used": common.MapStr{
				"enabled":                  h.ToBool("aof_enabled", info, errs, "persistence.used.enabled"),
				"rewrite_in_progress":      h.ToBool("aof_rewrite_in_progress", info, errs, "persistence.used.rewrite_in_progress"),
				"rewrite_scheduled":        h.ToBool("aof_rewrite_scheduled", info, errs, "persistence.used.rewrite_scheduled"),
				"last_rewrite_time_sec":    h.ToInt("aof_last_rewrite_time_sec", info, errs, "persistence.used.last_rewrite_time_sec"),
				"current_rewrite_time_sec": h.ToInt("aof_current_rewrite_time_sec", info, errs, "persistence.used.current_rewrite_time_sec"),
				"last_bgrewrite_status":    h.ToStr("aof_last_bgrewrite_status", info, errs, "persistence.used.last_bgrewrite_status"),
				"last_write_status":        h.ToStr("aof_last_write_status", info, errs, "persistence.used.last_write_status"),
			},
		},
		"replication": common.MapStr{
			"role":             h.ToStr("role", info, errs, "replication.role"),
			"connected_slaves": h.ToInt("connected_slaves", info, errs, "replication.connected_slaves"),
			"master_offset":    h.ToInt("master_repl_offset", info, errs, "replication.master_repl_offset"),
			"backlog": common.MapStr{
				"active":            h.ToInt("repl_backlog_active", info, errs, "replication.backlog.active"),
				"size":              h.ToInt("repl_backlog_size", info, errs, "replication.backlog.size"),
				"first_byte_offset": h.ToInt("repl_backlog_first_byte_offset", info, errs, "replication.backlog.first_byte_offset"),
				"histlen":           h.ToInt("repl_backlog_histlen", info, errs, "replication.backlog.histlen"),
			},
		},
		"server": common.MapStr{
			"version":          h.ToStr("redis_version", info, errs, "server.version"),
			"git_sha1":         h.ToStr("redis_git_sha1", info, errs, "server.git_sha1"),
			"git_dirty":        h.ToStr("redis_git_dirty", info, errs, "server.git_dirty"),
			"build_id":         h.ToStr("redis_build_id", info, errs, "server.build_id"),
			"mode":             h.ToStr("redis_mode", info, errs, "server.mode"),
			"os":               h.ToStr("os", info, errs, "server.os"),
			"arch_bits":        h.ToStr("arch_bits", info, errs, "server.arch_bits"),
			"multiplexing_api": h.ToStr("multiplexing_api", info, errs, "server.multiplexing_api"),
			"gcc_version":      h.ToStr("gcc_version", info, errs, "server.gcc_version"),
			"process_id":       h.ToInt("process_id", info, errs, "server.process_id"),
			"run_id":           h.ToStr("run_id", info, errs, "server.run_id"),
			"tcp_port":         h.ToInt("tcp_port", info, errs, "server.tcp_port"),
			"uptime":           h.ToInt("uptime_in_seconds", info, errs, "server.uptime"), // Uptime days was removed as duplicate
			"hz":               h.ToInt("hz", info, errs, "server.hz"),
			"lru_clock":        h.ToInt("lru_clock", info, errs, "server.lru_clock"),
			"config_file":      h.ToStr("config_file", info, errs, "server.config_file"),
		},
		"stats": common.MapStr{
			"connections": common.MapStr{
				"received": h.ToInt("total_connections_received", info, errs, "stats.connections.received"),
				"rejected": h.ToInt("rejected_connections", info, errs, "stats.connections.rejected"),
			},
			"total_commands_processed":  h.ToInt("total_commands_processed", info, errs, "stats.total_commands_processed"),
			"total_net_input_bytes":     h.ToInt("total_net_input_bytes", info, errs, "stats.total_net_input_bytes"),
			"total_net_output_bytes":    h.ToInt("total_net_output_bytes", info, errs, "stats.total_net_output_bytes"),
			"instantaneous_ops_per_sec": h.ToInt("instantaneous_ops_per_sec", info, errs, "stats.instantaneous_ops_per_sec"),
			"instantaneous_input_kbps":  h.ToFloat("instantaneous_input_kbps", info, errs, "stats.instantaneous_input_kbps"),
			"instantaneous_output_kbps": h.ToFloat("instantaneous_output_kbps", info, errs, "stats.instantaneous_output_kbps"),
			"sync": common.MapStr{
				"full":        h.ToInt("sync_full", info, errs, "stats.sync.full"),
				"partial_ok":  h.ToInt("sync_partial_ok", info, errs, "stats.sync.partial_ok"),
				"partial_err": h.ToInt("sync_partial_err", info, errs, "stats.sync.partial_err"),
			},
			"keys": common.MapStr{
				"expired": h.ToInt("expired_keys", info, errs, "stats.keys.expired"),
				"evicted": h.ToInt("evicted_keys", info, errs, "stats.keys.evicted"),
			},
			"keyspace": common.MapStr{
				"hits":   h.ToInt("keyspace_hits", info, errs, "stats.keyspace.hits"),
				"misses": h.ToInt("keyspace_misses", info, errs, "stats.keyspace.misses"),
			},
			"pubsub_channels":        h.ToInt("pubsub_channels", info, errs, "stats.pubsub_channels"),
			"pubsub_patterns":        h.ToInt("pubsub_patterns", info, errs, "stats.pubsub_patterns"),
			"latest_fork_usec":       h.ToInt("latest_fork_usec", info, errs, "stats.latest_fork_usec"),
			"migrate_cached_sockets": h.ToInt("migrate_cached_sockets", info, errs, "stats.migrate_cached_sockets"),
		},
	}
	h.RemoveErroredKeys(event, errs)

	return event
}
