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
		// Commonly known as performance counters
		"performance": s.Object{
			// Page splits -- cumulative per instance. Show diffs between periodic readings to identify periods of frequent page splits.
			"page_splits_seconds": c.Int("page_splits_sec"),
			// Page Life Expectancy -- the expected time in seconds that a data page will remain in the buffer pool
			"page_life_expectancy_seconds": c.Int("page_life_expectancy"),
			// Lock waits -- cumulative per instance. Show diffs between periodic readings to identify periods of high lock contention.
			"lock_waits_seconds": c.Int("lock_waits_sec"),
			"user_connections":   c.Int("user_connections"),
			// SQL re-compilations -- cumulative per instance. Show diffs between periodic readings to identify periods of high SQL re-compilations.
			"recompilations_seconds": c.Int("recompilations_sec"),
			// SQL compilations -- cumulative per instance. Show diffs between periodic readings to identify periods of high SQL compilations.
			"compilations_seconds": c.Int("compilations_sec"),
			// Transactions -- cumulative per database. Show diffs between periodic readings to identify periods of high transaction activity.
			"transactions_seconds": c.Int("transactions_sec"),
			// Batch requests -- cumulative per instance. Show diffs between periodic readings to identify periods of high request activity.
			"batch_requests_seconds": c.Int("batch_req_sec"),
			// Buffer cache hit ratio -- percentage of data pages found in buffer cache without having to read from disk
			"buffer_cache_hit_ratio": c.Float("buffer_cache_hit_ratio"),
		},
	}
)
