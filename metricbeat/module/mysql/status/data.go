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
