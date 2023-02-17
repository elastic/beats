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
)
