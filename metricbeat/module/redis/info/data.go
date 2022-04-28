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

package info

import (
	"strings"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	schema = s.Schema{
		"clients": s.Object{
			"connected":         c.Int("connected_clients"),
			"max_output_buffer": c.Int("client_recent_max_output_buffer"),
			"max_input_buffer":  c.Int("client_recent_max_input_buffer"),
			"blocked":           c.Int("blocked_clients"),
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
				"value":   c.Int("used_memory"), // As it is a top key, this goes into value
				"rss":     c.Int("used_memory_rss"),
				"peak":    c.Int("used_memory_peak"),
				"lua":     c.Int("used_memory_lua"),
				"dataset": c.Int("used_memory_dataset"),
			},
			"max": s.Object{
				"value":  c.Int("maxmemory"),
				"policy": c.Str("maxmemory_policy"),
			},
			"fragmentation": s.Object{
				"ratio": c.Float("mem_fragmentation_ratio"),
				"bytes": c.Int("mem_fragmentation_bytes"),
			},
			"active_defrag": s.Object{
				"is_running": c.Bool("active_defrag_running"),
			},
			"allocator": c.Str("mem_allocator"), // Could be moved to server as it rarely changes
			"allocator_stats": s.Object{
				"allocated": c.Int("allocator_allocated"),
				"active":    c.Int("allocator_active"),
				"resident":  c.Int("allocator_resident"),
				"fragmentation": s.Object{
					"ratio": c.Float("allocator_frag_ratio"),
					"bytes": c.Int("allocator_frag_bytes"),
				},
				"rss": s.Object{
					"ratio": c.Float("allocator_rss_ratio"),
					"bytes": c.Int("allocator_rss_bytes"),
				},
			},
		},
		"persistence": s.Object{
			"loading": c.Bool("loading"),
			"rdb": s.Object{
				"last_save": s.Object{
					"changes_since": c.Int("rdb_changes_since_last_save"),
					"time":          c.Int("rdb_last_save_time"),
				},
				"bgsave": s.Object{
					"last_status": c.Str("rdb_last_bgsave_status"),
					"in_progress": c.Bool("rdb_bgsave_in_progress"),
					"last_time": s.Object{
						"sec": c.Int("rdb_last_bgsave_time_sec"),
					},
					"current_time": s.Object{
						"sec": c.Int("rdb_current_bgsave_time_sec"),
					},
				},
				"copy_on_write": s.Object{
					"last_size": c.Int("rdb_last_cow_size"),
				},
			},
			"aof": s.Object{
				"enabled": c.Bool("aof_enabled"),
				"rewrite": s.Object{
					"in_progress": c.Bool("aof_rewrite_in_progress"),
					"scheduled":   c.Bool("aof_rewrite_scheduled"),
					"last_time": s.Object{
						"sec": c.Int("aof_last_rewrite_time_sec"),
					},
					"current_time": s.Object{
						"sec": c.Int("aof_current_rewrite_time_sec"),
					},
					"buffer": s.Object{
						"size": c.Int("aof_rewrite_buffer_length"),
					},
				},
				"bgrewrite": s.Object{
					"last_status": c.Str("aof_last_bgrewrite_status"),
				},
				"write": s.Object{
					"last_status": c.Str("aof_last_write_status"),
				},
				"copy_on_write": s.Object{
					"last_size": c.Int("aof_last_cow_size"),
				},
				"buffer": s.Object{
					"size": c.Int("aof_buffer_length"),
				},
				"size": s.Object{
					"current": c.Int("aof_current_size"),
					"base":    c.Int("aof_base_size"),
				},
				"fsync": s.Object{
					"pending": c.Int("aof_pending_bio_fsync"),
					"delayed": c.Int("aof_delayed_fsync"),
				},
			},
		},
		"replication": s.Object{
			"role":             c.Str("role"),
			"connected_slaves": c.Int("connected_slaves"),
			"backlog": s.Object{
				"active":            c.Int("repl_backlog_active"),
				"size":              c.Int("repl_backlog_size"),
				"first_byte_offset": c.Int("repl_backlog_first_byte_offset"),
				"histlen":           c.Int("repl_backlog_histlen"),
			},
			"master": s.Object{
				"offset":              c.Int("master_repl_offset"),
				"second_offset":       c.Int("second_repl_offset"),
				"link_status":         c.Str("master_link_status", s.Optional),
				"last_io_seconds_ago": c.Int("master_last_io_seconds_ago", s.Optional),
				"sync": s.Object{
					"in_progress":         c.Bool("master_sync_in_progress", s.Optional),
					"left_bytes":          c.Int("master_sync_left_bytes", s.Optional),
					"last_io_seconds_ago": c.Int("master_sync_last_io_seconds_ago", s.Optional),
				},
			},
			"slave": s.Object{
				"offset":      c.Int("slave_repl_offset", s.Optional),
				"priority":    c.Int("slave_priority", s.Optional),
				"is_readonly": c.Bool("slave_read_only", s.Optional),
			},
			// ToDo find a way to add dynamic object of slaves: "slaves": s.Str("slaveXXX")
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
			"net": s.Object{
				"input": s.Object{
					"bytes": c.Int("total_net_input_bytes"),
				},
				"output": s.Object{
					"bytes": c.Int("total_net_output_bytes"),
				},
			},
			"instantaneous": s.Object{
				"ops_per_sec": c.Int("instantaneous_ops_per_sec"),
				"input_kbps":  c.Float("instantaneous_input_kbps"),
				"output_kbps": c.Float("instantaneous_output_kbps"),
			},
			"sync": s.Object{
				"full": c.Int("sync_full"),
				"partial": s.Object{
					"ok":  c.Int("sync_partial_ok"),
					"err": c.Int("sync_partial_err"),
				},
			},
			"keys": s.Object{
				"expired": c.Int("expired_keys"),
				"evicted": c.Int("evicted_keys"),
			},
			"keyspace": s.Object{
				"hits":   c.Int("keyspace_hits"),
				"misses": c.Int("keyspace_misses"),
			},
			"pubsub": s.Object{
				"channels": c.Int("pubsub_channels"),
				"patterns": c.Int("pubsub_patterns"),
			},
			"latest_fork_usec":           c.Int("latest_fork_usec"),
			"migrate_cached_sockets":     c.Int("migrate_cached_sockets"),
			"slave_expires_tracked_keys": c.Int("slave_expires_tracked_keys"),
			"active_defrag": s.Object{
				"hits":       c.Int("active_defrag_hits"),
				"misses":     c.Int("active_defrag_misses"),
				"key_hits":   c.Int("active_defrag_key_hits"),
				"key_misses": c.Int("active_defrag_key_misses"),
			},
		},
		"slowlog": s.Object{
			"count": c.Int("slowlog_len"),
		},
	}
)

func buildCommandstatsSchema(key string, schema s.Schema) {
	// Build schema for each command
	command := strings.Split(key, "_")[1]
	schema[command] = s.Object{
		"calls":          c.Int("cmdstat_" + command + "_calls"),
		"usec":           c.Int("cmdstat_" + command + "_usec"),
		"usec_per_call":  c.Float("cmdstat_" + command + "_usec_per_call"),
		"rejected_calls": c.Int("cmdstat_" + command + "_rejected_calls"),
		"failed_calls":   c.Int("cmdstat_" + command + "_failed_calls"),
	}
}

// Map data to MapStr
func eventMapping(r mb.ReporterV2, info map[string]string) {
	// Full mapping from info
	source := map[string]interface{}{}
	commandstatsSchema := s.Schema{}
	for key, val := range info {
		source[key] = val
		if strings.HasPrefix(key, "cmdstat_") {
			buildCommandstatsSchema(key, commandstatsSchema)
		}
	}
	data, _ := schema.Apply(source)

	// Add commandstats info
	commandstatsData, _ := commandstatsSchema.Apply(source)
	data["commandstats"] = commandstatsData

	rootFields := mapstr.M{}
	if v, err := data.GetValue("server.version"); err == nil {
		rootFields.Put("service.version", v)
		data.Delete("server.version")
	}
	if v, err := data.GetValue("server.process_id"); err == nil {
		rootFields.Put("process.pid", v)
		data.Delete("server.process_id")
	}
	if v, err := data.GetValue("server.os"); err == nil {
		rootFields.Put("os.full", v)
		data.Delete("server.os")
	}
	r.Event(mb.Event{
		MetricSetFields: data,
		RootFields:      rootFields,
	})
}
