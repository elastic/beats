package status

import (
	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

var (
	schema = h.NewSchema(common.MapStr{
		"aborted": common.MapStr{
			"clients":  h.Int("Aborted_clients"),
			"connects": h.Int("Aborted_connects"),
		},
		"binlog": common.MapStr{
			"cache": common.MapStr{
				"disk_use": h.Int("Binlog_cache_disk_use"),
				"use":      h.Int("Binlog_cache_use"),
			},
		},
		"bytes": common.MapStr{
			"received": h.Int("Bytes_received"),
			"sent":     h.Int("Bytes_sent"),
		},
		"connections": h.Int("Connections"),
		"created": common.MapStr{
			"tmp": common.MapStr{
				"disk_tables": h.Int("Created_tmp_disk_tables"),
				"files":       h.Int("Created_tmp_files"),
				"tables":      h.Int("Created_tmp_tables"),
			},
		},
		"delayed": common.MapStr{
			"errors":         h.Int("Delayed_errors"),
			"insert_threads": h.Int("Delayed_insert_threads"),
			"writes":         h.Int("Delayed_writes"),
		},
		"flush_commands":       h.Int("Flush_commands"),
		"max_used_connections": h.Int("Max_used_connections"),
		"open": common.MapStr{
			"files":   h.Int("Open_files"),
			"streams": h.Int("Open_streams"),
			"tables":  h.Int("Open_tables"),
		},
		"opened_tables": h.Int("Opened_tables"),
	})
)

// Map data to MapStr of server stats variables: http://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html
// This is only a subset of the available values
func eventMapping(status map[string]string) common.MapStr {
	return schema.Apply(status)
}
