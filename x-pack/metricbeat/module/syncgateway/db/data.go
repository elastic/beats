// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package db

import (
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/syncgateway"
)

type SgResponse struct {
	SyncgatewayChangeCache struct {
		MaxPending float64 `json:"maxPending"`
	} `json:"syncGateway_changeCache"`
	Syncgateway Syncgateway            `json:"syncgateway"`
	MemStats    map[string]interface{} `json:"memstats"`
}

type Syncgateway struct {
	Global struct {
		ResourceUtilization map[string]interface{} `json:"resource_utilization"`
	} `json:"global"`
	PerDb          map[string]map[string]interface{} `json:"per_db"`
	PerReplication map[string]map[string]interface{} `json:"per_replication"`
}

var (
	dbSchema = s.Schema{
		"cache": c.Dict("cache", s.Schema{
			"channel": s.Object{
				"revs": s.Object{
					"active":    c.Float("chan_cache_active_revs"),
					"removal":   c.Float("chan_cache_removal_revs"),
					"tombstone": c.Float("chan_cache_tombstone_revs"),
				},
				"hits":   c.Float("chan_cache_hits"),
				"misses": c.Float("chan_cache_misses"),
			},
			"revs": s.Object{
				"hits":   c.Float("rev_cache_hits"),
				"misses": c.Float("rev_cache_misses"),
			},
		}),
		"metrics": c.Dict("database", s.Schema{
			"replications": s.Object{
				"active": c.Float("num_replications_active"),
				"total":  c.Float("num_replications_total"),
			},
			"docs": s.Object{
				"writes": s.Object{
					"count":    c.Float("num_doc_writes"),
					"bytes":    c.Float("doc_writes_bytes"),
					"conflict": s.Object{"count": c.Float("conflict_write_count")},
				},
			},
		}),
		"security": c.Dict("security", s.Schema{
			"auth": s.Object{
				"failed": s.Object{"count": c.Float("auth_failed_count")},
			},
			"access_errors": s.Object{"count": c.Float("num_access_errors")},
			"docs_rejected": s.Object{"count": c.Float("num_docs_rejected")},
		}),
		"cbl": s.Object{
			"replication": s.Object{
				"pull": c.Dict("cbl_replication_pull", s.Schema{
					"attachment": s.Object{
						"bytes": c.Float("attachment_pull_bytes"),
						"count": c.Float("attachment_pull_count"),
					},
					"active": s.Object{
						"count":      c.Float("num_replications_active"),
						"continuous": c.Float("num_pull_repl_active_continuous"),
						"one_shot":   c.Float("num_pull_repl_active_one_shot"),
					},
					"total": s.Object{
						"continuous": c.Float("num_pull_repl_total_continuous"),
						"one_shot":   c.Float("num_pull_repl_total_one_shot"),
					},
					"caught_up":  c.Float("num_pull_repl_caught_up"),
					"since_zero": c.Float("num_pull_repl_since_zero"),
					"request_changes": s.Object{
						"count": c.Float("request_changes_count"),
						"time":  c.Float("request_changes_time"),
					},
					"rev": s.Object{
						"processing_time": c.Float("rev_processing_time"),
						"send": s.Object{
							"count":   c.Float("rev_send_count"),
							"latency": c.Float("rev_send_latency"),
						},
					},
				}),
				"push": c.Dict("cbl_replication_push", s.Schema{
					"write_processing_time": c.Float("write_processing_time"),
					"doc_push_count":        c.Float("doc_push_count"),
					"attachment": s.Object{
						"bytes": c.Float("attachment_push_bytes"),
						"count": c.Float("attachment_push_count"),
					},
					"propose_change": s.Object{
						"count": c.Float("propose_change_count"),
						"time":  c.Float("propose_change_time"),
					},
					"sync_function": s.Object{
						"count": c.Float("sync_function_count"),
						"time":  c.Float("sync_function_time"),
					},
				}),
			},
		},
		"gsi": s.Object{
			"views": c.Dict("gsi_views", s.Schema{
				"access": s.Object{
					"query": s.Object{
						"count": c.Float("access_query_count"),
						"error": s.Object{"count": c.Float("access_query_error_count")},
						"time":  c.Float("access_query_time"),
					},
				},
				"all_docs": s.Object{
					"query": s.Object{
						"count": c.Float("allDocs_query_count"),
						"error": s.Object{"count": c.Float("allDocs_query_error_count")},
						"time":  c.Float("allDocs_query_time"),
					},
				},
				"channels": s.Object{
					"star": s.Object{
						"query": s.Object{
							"count": c.Float("channelsStar_query_count"),
							"error": s.Object{"count": c.Float("channelsStar_query_error_count")},
							"time":  c.Float("channelsStar_query_time"),
						},
					},
					"query": s.Object{
						"count": c.Float("channels_query_count"),
						"error": s.Object{"count": c.Float("channels_query_error_count")},
						"time":  c.Float("channels_query_time"),
					},
				},
				"principals": s.Object{
					"query": s.Object{
						"count": c.Float("principals_query_count"),
						"error": s.Object{"count": c.Float("principals_query_error_count")},
						"time":  c.Float("principals_query_time"),
					},
				},
				"resync": s.Object{
					"query": s.Object{
						"count": c.Float("resync_query_count"),
						"error": s.Object{"count": c.Float("resync_query_error_count")},
						"time":  c.Float("resync_query_time"),
					},
				},
				"role_access": s.Object{
					"query": s.Object{
						"count": c.Float("roleAccess_query_count"),
						"error": s.Object{"count": c.Float("roleAccess_query_error_count")},
						"time":  c.Float("roleAccess_query_time"),
					},
				},
				"sequences": s.Object{
					"query": s.Object{
						"count": c.Float("sequences_query_count"),
						"error": s.Object{"count": c.Float("sequences_query_error_count")},
						"time":  c.Float("sequences_query_time"),
					},
				},
				"sessions": s.Object{
					"query": s.Object{
						"count": c.Float("sessions_query_count"),
						"error": s.Object{"count": c.Float("sessions_query_error_count")},
						"time":  c.Float("sessions_query_time"),
					},
				},
				"tombstones": s.Object{
					"query": s.Object{
						"count": c.Float("tombstones_query_count"),
						"error": s.Object{"count": c.Float("tombstones_query_error_count")},
						"time":  c.Float("tombstones_query_time"),
					},
				},
			}),
		},
	}
)

func eventMapping(r mb.ReporterV2, content *syncgateway.SgResponse) {
	for dbName, db := range content.Syncgateway.PerDb {
		dbData, _ := dbSchema.Apply(db)
		dbData.Put("name", dbName)
		r.Event(mb.Event{
			MetricSetFields: dbData,
		})
	}
}
