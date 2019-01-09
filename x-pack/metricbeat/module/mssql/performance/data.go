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
		"page_life_expectancy": s.Object{
			"sec": c.Int("Page life expectancy", s.Optional),
		},
		"lock_waits": s.Object{
			"sec": c.Int("Lock Waits/sec", s.Optional),
		},
		"user_connections": c.Int("User Connections", s.Optional),
		"recompilations": s.Object{
			"sec": c.Int("SQL Re-Compilations/sec", s.Optional),
		},
		"compilations": s.Object{
			"sec": c.Int("SQL Compilations/sec", s.Optional),
		},
		"batch_requests": s.Object{
			"sec": c.Int("Batch Requests/sec", s.Optional),
		},
		"buffer_cache_hit": s.Object{
			"pct": c.Float("Buffer cache hit ratio", s.Optional),
		},
	}
)
