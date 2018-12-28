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
			"sec": c.Int("page_splits_sec"),
		},
		"page_life_expectancy": s.Object{
			"sec": c.Int("page_life_expectancy"),
		},
		"lock_waits": s.Object{
			"sec": c.Int("lock_waits_sec"),
		},
		"user_connections": c.Int("user_connections"),
		"recompilations": s.Object{
			"sec": c.Int("recompilations_sec"),
		},
		"compilations": s.Object{
			"sec": c.Int("compilations_sec"),
		},
		"transactions": s.Object{
			"sec": c.Int("transactions_sec"),
		},
		"batch_requests": s.Object{
			"sec": c.Int("batch_req_sec"),
		},
		"buffer_cache_hit_ratio": c.Float("buffer_cache_hit_ratio"),
	}
)
