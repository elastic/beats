/*
Package info fetches Redis server information and statistics using the Redis
INFO command.

The current implementation is tested with redis 3.0.7
More details on all the fields provided by the redis info command can be found here: http://redis.io/commands/INFO

`info.go` uses the Redis `INFO default` command for stats. This allows us to fetch  all metrics at once and filter out
undesired metrics based on user configuration on the client. The alternative would be to fetch each type as an
independent `INFO` call, which has the potential of introducing higher latency (e.g., more round trip Redis calls).

There's a special case for the `keyspace` stat, which requires additional parsing because the
values are overloaded: (keys=795341,expires=0,avg_ttl=0). This is handled in `eventMapping()`.

The document sent to elasticsearch has the following structure:

	{
	  "metricset": "info",
	  "module": "redis",
	  "redis-info": {
	    "clients": {
	      "blocked_clients": "0",
	      "client_biggest_input_buf": "0",
	      "client_longest_output_list": "0",
	      "connected_clients": "3"
	    },
	    "cluster": {
	      "cluster_enabled": "0"
	    },
	    "cpu": {
	      "used_cpu_sys": "210.63",
	      "used_cpu_sys_children": "0.00",
	      "used_cpu_user": "113.11",
	      "used_cpu_user_children": "0.00"
	    },
	    "memory": {
	      "mem_allocator": "libc",
	      "used_memory": "1043200",
	      "used_memory_lua": "36864",
	      "used_memory_peak": "1164080",
	      "used_memory_rss": "778240"
	    },
	    "persistence": {
	      "aof_current_rewrite_time_sec": "-1",
	      "aof_enabled": "0",
	      "aof_last_bgrewrite_status": "ok",
	      "aof_last_rewrite_time_sec": "-1",
	      "aof_last_write_status": "ok",
	      "aof_rewrite_in_progress": "0",
	      "aof_rewrite_scheduled": "0",
	      "loading": "0",
	      "rdb_bgsave_in_progress": "0",
	      "rdb_changes_since_last_save": "1",
	      "rdb_current_bgsave_time_sec": "-1",
	      "rdb_last_bgsave_status": "ok",
	      "rdb_last_bgsave_time_sec": "0",
	      "rdb_last_save_time": "1456758970"
	    },
	    "replication": {
	      "connected_slaves": "0",
	      "master_repl_offset": "0",
	      "repl_backlog_active": "0",
	      "repl_backlog_first_byte_offset": "0",
	      "repl_backlog_histlen": "0",
	      "repl_backlog_size": "1048576",
	      "role": "master"
	    },
	    "server": {
	      "arch_bits": "64",
	      "config_file": "",
	      "gcc_version": "4.2.1",
	      "hz": "10",
	      "lru_clock": "13918572",
	      "multiplexing_api": "kqueue",
	      "os": "Darwin 15.3.0 x86_64",
	      "process_id": "1158",
	      "redis_build_id": "aa27a151289c9b98",
	      "redis_git_dirty": "0",
	      "redis_git_sha1": "00000000",
	      "redis_mode": "standalone",
	      "redis_version": "3.0.7",
	      "run_id": "8e1659f076c248591812705a24e545257ee6e090",
	      "tcp_port": "6379",
	      "uptime_in_days": "20",
	      "uptime_in_seconds": "1730008"
	    },
	    "stats": {
	      "evicted_keys": "0",
	      "expired_keys": "0",
	      "instantaneous_input_kbps": "0.01",
	      "instantaneous_ops_per_sec": "0",
	      "instantaneous_output_kbps": "1.16",
	      "keyspace_hits": "1",
	      "keyspace_misses": "0",
	      "latest_fork_usec": "376",
	      "migrate_cached_sockets": "0",
	      "pubsub_channels": "0",
	      "pubsub_patterns": "0",
	      "rejected_connections": "0",
	      "sync_full": "0",
	      "sync_partial_err": "0",
	      "sync_partial_ok": "0",
	      "total_commands_processed": "151",
	      "total_connections_received": "146",
	      "total_net_input_bytes": "2247",
	      "total_net_output_bytes": "277354"
	    },
	    "keyspace": {
	      "db0": {
	        "avg_ttl": 0,
	    	"expires": 0,
	    	"keys": 1
	      },
	      "db1": {
	    	"avg_ttl": 0,
	    	"expires": 0,
	    	"keys": 1
	      }
	    }
	  }
	}
*/
package info
