package database


import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"active_transaction":       c.Int("Transactions", s.Optional),
		"deadlock_count_total": c.Int("Number of Deadlocks/sec", s.Optional),
		"lock_request_total": c.Int("Lock Requests/sec", s.Optional),
		"table_full_scan_count": c.Int("Full Scans/sec", s.Optional),
		"plan_cache_hit_ratio": c.Float("PlanCacheHitRatio", s.Optional),
		"buffer_page_fault": c.Int("PageFaultCount", s.Optional),
	}

	tableSpaceSchema = s.Schema{
		"used_space": c.Int("used_space", s.Optional),
		"unused_space": c.Int("unused_space", s.Optional),
		"space_used_pct": c.Float("space_used_pct", s.Optional),
		"table_total_space": c.Int("table_total_space", s.Optional),
		"table_used_space": c.Int("table_used_space", s.Optional),
		"table_unused_space": c.Int("table_unused_space", s.Optional),
		"table_space_used_pct": c.Float("table_space_used_pct", s.Optional),
	}

	tableIndexSchema = s.Schema{
		"index_size": c.Int("index_size", s.Optional),
		"table_index_size": c.Int("table_index_size", s.Optional),
	}

	tableLogSchema = s.Schema{
		"log_size": c.Int("log_size", s.Optional),
	}

	ioWaitSchema = s.Schema{
		"io_wait": c.Float("io_wait", s.Optional),
	}

	diskReadWriteBytesSchema = s.Schema{
		"disk_input": c.Int("disk_input", s.Optional),
		"disk_output": c.Int("disk_output", s.Optional),
		"disk_io_avg_milli_second": c.Float("disk_io_avg_milli_second", s.Optional),
	}

	databaseNetworkSchema = s.Schema{
		"network_input_bytes": c.Int("input_bytes", s.Optional),
		"network_output_bytes": c.Int("network_output_bytes", s.Optional),
	}

	databaseSessionSchema = s.Schema{
		"session_block_count": c.Int("session_block_count", s.Optional),
	}
)
