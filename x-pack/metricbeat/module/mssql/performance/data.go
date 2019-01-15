// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"page_splits": s.Object{
			"sec": c.Int("Page Splits/sec", s.Optional),
		},
		"lock_waits": s.Object{
			"sec": c.Int("Lock Waits/sec", s.Optional),
		},
		"user_connections":   c.Int("User Connections", s.Optional),
		"transactions":       c.Int("Transactions", s.Optional),
		"active_temp_tables": c.Int("Active Temp Tables", s.Optional),
		"connections_reset": s.Object{
			"sec": c.Int("Connection Reset/sec", s.Optional),
		},
		"logouts": s.Object{
			"sec": c.Int("Logouts/sec", s.Optional),
		},
		"logins": s.Object{
			"sec": c.Int("Logins/sec", s.Optional),
		},
		"recompilations": s.Object{
			"sec": c.Int("SQL Re-Compilations/sec", s.Optional),
		},
		"compilations": s.Object{
			"sec": c.Int("SQL Compilations/sec", s.Optional),
		},
		"batch_requests": s.Object{
			"sec": c.Int("Batch Requests/sec", s.Optional),
		},
		"buffer": s.Object{
			"cache_hit": s.Object{
				"pct": c.Float("Buffer cache hit ratio", s.Optional),
			},
			"page_life_expectancy": s.Object{
				"sec": c.Int("Page life expectancy", s.Optional),
			},
			"checkpoint_pages": s.Object{
				"sec": c.Int("Checkpoint pages/sec", s.Optional),
			},
			"database_pages": c.Int("Database pages", s.Optional),
			"target_pages":   c.Int("Target pages", s.Optional),
		},
	}
)
