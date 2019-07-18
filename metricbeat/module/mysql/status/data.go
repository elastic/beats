// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package status

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"aborted": s.Object{
			"clients":  c.Int("Aborted_clients"),
			"connects": c.Int("Aborted_connects"),
		},
		"binlog": s.Object{
			"cache": s.Object{
				"disk_use": c.Int("Binlog_cache_disk_use"),
				"use":      c.Int("Binlog_cache_use"),
			},
		},
		"bytes": s.Object{
			"received": c.Int("Bytes_received"),
			"sent":     c.Int("Bytes_sent"),
		},
		"threads": s.Object{
			"cached":    c.Int("Threads_cached"),
			"created":   c.Int("Threads_created"),
			"connected": c.Int("Threads_connected"),
			"running":   c.Int("Threads_running"),
		},
		"connections": c.Int("Connections"),
		"created": s.Object{
			"tmp": s.Object{
				"disk_tables": c.Int("Created_tmp_disk_tables"),
				"files":       c.Int("Created_tmp_files"),
				"tables":      c.Int("Created_tmp_tables"),
			},
		},
		"delayed": s.Object{
			"errors":         c.Int("Delayed_errors"),
			"insert_threads": c.Int("Delayed_insert_threads"),
			"writes":         c.Int("Delayed_writes"),
		},
		"flush_commands":       c.Int("Flush_commands"),
		"max_used_connections": c.Int("Max_used_connections"),
		"open": s.Object{
			"files":   c.Int("Open_files"),
			"streams": c.Int("Open_streams"),
			"tables":  c.Int("Open_tables"),
		},
		"opened_tables": c.Int("Opened_tables"),
		"command": s.Object{
			"delete": c.Int("Com_delete"),
			"insert": c.Int("Com_insert"),
			"select": c.Int("Com_select"),
			"update": c.Int("Com_update"),
		},
		"queries":   c.Int("Queries"),
		"questions": c.Int("Questions"),
		"handler": s.Object{
			"commit":        c.Int("Handler_commit"),
			"delete":        c.Int("Handler_delete"),
			"external_lock": c.Int("Handler_external_lock"),
			"mrr_init":      c.Int("Handler_mrr_init"),
			"prepare":       c.Int("Handler_prepare"),
			"read": s.Object{
				"first":    c.Int("Handler_read_first"),
				"key":      c.Int("Handler_read_key"),
				"last":     c.Int("Handler_read_last"),
				"next":     c.Int("Handler_read_next"),
				"prev":     c.Int("Handler_read_prev"),
				"rnd":      c.Int("Handler_read_rnd"),
				"rnd_next": c.Int("Handler_read_rnd_next"),
			},
			"rollback":           c.Int("Handler_rollback"),
			"savepoint":          c.Int("Handler_savepoint"),
			"savepoint_rollback": c.Int("Handler_savepoint_rollback"),
			"update":             c.Int("Handler_update"),
			"write":              c.Int("Handler_write"),
		},
		"innodb": s.Object{
			"buffer_pool": s.Object{
				"dump_status": c.Int("Innodb_buffer_pool_dump_status"),
				"load_status": c.Int("Innodb_buffer_pool_load_status"),
				"bytes": s.Object{
					"data":  c.Int("Innodb_buffer_pool_bytes_data"),
					"dirty": c.Int("Innodb_buffer_pool_bytes_dirty"),
				},
				"pages": s.Object{
					"data":    c.Int("Innodb_buffer_pool_pages_data"),
					"dirty":   c.Int("Innodb_buffer_pool_pages_dirty"),
					"flushed": c.Int("Innodb_buffer_pool_pages_flushed"),
					"free":    c.Int("Innodb_buffer_pool_pages_free"),
					"latched": c.Int("Innodb_buffer_pool_pages_latched"),
					"misc":    c.Int("Innodb_buffer_pool_pages_misc"),
					"total":   c.Int("Innodb_buffer_pool_pages_total"),
				},
				"read": s.Object{
					"ahead":         c.Int("Innodb_buffer_pool_read_ahead"),
					"ahead_evicted": c.Int("Innodb_buffer_pool_read_ahead_evicted"),
					"ahead_rnd":     c.Int("Innodb_buffer_pool_read_ahead_rnd"),
					"requests":      c.Int("Innodb_buffer_pool_read_requests"),
				},
				"pool": s.Object{
					"reads":         c.Int("Innodb_buffer_pool_reads"),
					"resize_status": c.Int("Innodb_buffer_pool_resize_status"),
					"wait_free":     c.Int("Innodb_buffer_pool_wait_free"),
				},
				"write_requests": c.Int("Innodb_buffer_pool_write_requests"),
			},
		},
	}
)

// Map data to MapStr of server stats variables: http://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html
// This is only a subset of the available values
func eventMapping(status map[string]string) common.MapStr {
	source := map[string]interface{}{}
	for key, val := range status {
		source[key] = val
	}
	data, _ := schema.Apply(source)
	return data
}

func rawEventMapping(status map[string]string) common.MapStr {
	source := common.MapStr{}
	for key, val := range status {
		// Only adds events which are not in the mapping
		if schema.HasKey(key) {
			continue
		}

		source[key] = val
	}
	return source
}
