// +build integration

package os

import (
	"time"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		// Returns a miscellaneous set of useful information about the computer, and about the resources available to and consumed by SQL Server.
		"info": s.Object{
			"cpu_ticks":                      c.Int("cpu_ticks"),
			"ms_ticks":                       c.Int("ms_ticks"),
			"cpu_count":                      c.Int("cpu_count"),
			"hyperthread_ratio":              c.Int("hyperthread_ratio"),
			"physical_memory_in_bytes":       c.Int("physical_memory_in_bytes"),
			"physical_memory_kb":             c.Int("physical_memory_kb"),
			"virtual_memory_in_bytes":        c.Int("virtual_memory_in_bytes"),
			"virtual_memory_kb":              c.Int("virtual_memory_kb"),
			"bpool_commited":                 c.Int("bpool_commited"),
			"committed_kb":                   c.Int("committed_kb"),
			"bpool_commit_target":            c.Int("bpool_commit_target"),
			"committed_target_kb":            c.Int("committed_target_kb"),
			"bpool_visible":                  c.Int("bpool_visible"),
			"visible_target_kb":              c.Int("visible_target_kb"),
			"stack_size_in_bytes":            c.Int("stack_size_in_bytes"),
			"os_quantum":                     c.Int("os_quantum"),
			"os_error_mode":                  c.Int("os_error_mode"),
			"os_priority_class":              c.Int("os_priority_class"),
			"max_workers_count":              c.Int("max_workers_count"),
			"scheduler_total_count":          c.Int("scheduler_total_count"),
			"deadlock_monitor_serial_number": c.Int("deadlock_monitor_serial_number"),
			"sqlserver_start_time_ms_ticks":  c.Int("sqlserver_start_time_ms_ticks"),
			"sqlserver_start_time":           c.Time(time.RFC3339, "sqlserver_start_time"),
			"affinity_type":                  c.Int("affinity_type"),
			"affinity_type_desc":             c.Str("affinity_type_desc"),
			"process_kernel_time_ms":         c.Int("process_kernel_time_ms"),
			"process_user_time_ms":           c.Int("process_user_time_ms"),
			"time_source_desc":               c.Str("time_source_desc"),
			"virtual_machine_type":           c.Int("virtual_machine_type"),
			"virtual_machine_type_desc":      c.Str("virtual_machine_type_desc"),
			"softnuma_configuration":         c.Int("softnuma_configuration"),
			"softnuma_configuration_desc":    c.Str("softnuma_configuration_desc"),
			"process_physical_affinity":      c.Str("process_physical_affinity"),
			"sql_memory_model":               c.Int("sql_memory_model"),
			"sql_memory_model_desc":          c.Str("sql_memory_model_desc"),
			"socket_count":                   c.Int("socket_count"),
			"cores_per_socket":               c.Int("cores_per_socket"),
			"numa_node_count":                c.Int("numa_node_count"),
			// SQL Server uptime to discover unexpected reboots or restarts.
			"uptime_seconds": c.Int("uptime_seconds"),
		},
		// Returns memory information from the operating system.
		"memory": s.Object{
			"total_physical_memory_kb":        c.Int("total_physical_memory_kb"),
			"available_physical_memory_kb":    c.Int("available_physical_memory_kb"),
			"total_page_file_kb":              c.Int("total_page_file_kb"),
			"available_page_file_kb":          c.Int("available_page_file_kb"),
			"system_cache_kb":                 c.Int("system_cache_kb"),
			"kernel_paged_pool_kb":            c.Int("kernel_paged_pool_kb"),
			"kernel_nonpaged_pool_kb":         c.Int("kernel_nonpaged_pool_kb"),
			"system_high_memory_signal_state": c.Int("system_high_memory_signal_state"),
			"system_low_memory_signal_state":  c.Int("system_low_memory_signal_state"),
			"system_memory_state_desc":        c.Str("system_memory_state_desc"),
			"pdw_node_id":                     c.Int("pdw_node_id"),
		},
		"virtualfilestats": s.Object{
			"db_name":                     c.Str("db_name"),
			"io_stall_read_milliseconds":  c.Int("io_stall_read_milliseconds"),
			"io_stall_write_milliseconds": c.Int("io_stall_write_milliseconds"),
		},
	}
)
