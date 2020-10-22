// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var (
	schemaCPUUsage = s.Schema{
		"total_usage":         c.Int("total_usage"),
		"usage_in_kernelmode": c.Int("usage_in_kernelmode"),
		"usage_in_usermode":   c.Int("usage_in_usermode"),
	}

	schemaThrottlingData = s.Schema{
		"periods":           c.Int("periods"),
		"throttled_periods": c.Int("throttled_periods"),
		"throttled_time":    c.Int("throttled_time"),
	}

	schemaMemoryStatsStats = s.Schema{
		"active_anon":               c.Int("active_anon"),
		"active_file":               c.Int("active_file"),
		"cache":                     c.Int("cache"),
		"dirty":                     c.Int("dirty"),
		"hierarchical_memory_limit": c.Int("hierarchical_memory_limit"),
		"hierarchical_memsw_limit":  c.Int("hierarchical_memsw_limit"),
		"inactive_anon":             c.Int("inactive_anon"),
		"inactive_file":             c.Int("inactive_file"),
		"mapped_file":               c.Int("mapped_file"),
		"pgfault":                   c.Int("pgfault"),
		"pgmajfault":                c.Int("pgmajfault"),
		"pgpgin":                    c.Int("pgpgin"),
		"pgpgout":                   c.Int("pgpgout"),
		"rss":                       c.Int("rss"),
		"rss_huge":                  c.Int("rss_huge"),
		"total_active_anon":         c.Int("total_active_anon"),
		"total_active_file":         c.Int("total_active_file"),
		"total_cache":               c.Int("total_cache"),
		"total_dirty":               c.Int("total_dirty"),
		"total_inactive_anon":       c.Int("total_inactive_anon"),
		"total_inactive_file":       c.Int("total_inactive_file"),
		"total_mapped_file":         c.Int("total_mapped_file"),
		"total_pgfault":             c.Int("total_pgfault"),
		"total_pgmajfault":          c.Int("total_pgmajfault"),
		"total_pgpgin":              c.Int("total_pgpgin"),
		"total_pgpgout":             c.Int("total_pgpgout"),
		"total_rss":                 c.Int("total_rss"),
		"total_rss_huge":            c.Int("total_rss_huge"),
		"total_unevictable":         c.Int("total_unevictable"),
		"total_writeback":           c.Int("total_writeback"),
		"unevictable":               c.Int("unevictable"),
		"writeback":                 c.Int("writeback"),
	}

	schemaNetwork = s.Schema{
		"rx_bytes":   c.Int("rx_bytes"),
		"rx_packets": c.Int("rx_packets"),
		"rx_errors":  c.Int("rx_errors"),
		"rx_dropped": c.Int("rx_dropped"),
		"tx_bytes":   c.Int("tx_bytes"),
		"tx_packets": c.Int("tx_packets"),
		"tx_errors":  c.Int("tx_errors"),
		"tx_dropped": c.Int("tx_dropped"),
	}

	schemaCPUStats = s.Schema{
		"cpu_stats": s.Object{
			"cpu_usage":       c.Dict("cpu_usage", schemaCPUUsage),
			"throttling_data": c.Dict("throttling_data", schemaThrottlingData),
		},
	}

	schemaMemoryStats = s.Schema{
		"memory_stats": s.Object{
			"usage":     c.Int("total_usage"),
			"max_usage": c.Int("max_usage"),
			"limit":     c.Int("limit"),
			"stats":     c.Dict("stats", schemaMemoryStatsStats),
		},
	}
)
