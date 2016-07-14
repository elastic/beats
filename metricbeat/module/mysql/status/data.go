package status

import (
	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

// Map data to MapStr of server stats variables: http://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html
// This is only a subset of the available values
func eventMapping(status map[string]string) common.MapStr {

	errs := map[string]error{}
	event := common.MapStr{
		"aborted": common.MapStr{
			"clients":  h.ToInt("Aborted_clients", status, errs, "aborted.clients"),
			"connects": h.ToInt("Aborted_connects", status, errs, "aborted.connects"),
		},
		"binlog": common.MapStr{
			"cache": common.MapStr{
				"disk_use": h.ToInt("Binlog_cache_disk_use", status, errs, "binlog.cache.disk_use"),
				"use":      h.ToInt("Binlog_cache_use", status, errs, "binlog.cache.use"),
			},
		},
		"bytes": common.MapStr{
			"received": h.ToInt("Bytes_received", status, errs, "bytes.received"),
			"sent":     h.ToInt("Bytes_sent", status, errs, "bytes.sent"),
		},
		"connections": h.ToInt("Connections", status, errs, "connections"),
		"created": common.MapStr{
			"tmp": common.MapStr{
				"disk_tables": h.ToInt("Created_tmp_disk_tables", status, errs, "created.tmp.disk_tables"),
				"files":       h.ToInt("Created_tmp_files", status, errs, "created.tmp.files"),
				"tables":      h.ToInt("Created_tmp_tables", status, errs, "created.tmp.tables"),
			},
		},
		"delayed": common.MapStr{
			"errors":         h.ToInt("Delayed_errors", status, errs, "delayed.errors"),
			"insert_threads": h.ToInt("Delayed_insert_threads", status, errs, "delayed.insert_threads"),
			"writes":         h.ToInt("Delayed_writes", status, errs, "delayed.writes"),
		},
		"flush_commands":       h.ToInt("Flush_commands", status, errs, "flush_commands"),
		"max_used_connections": h.ToInt("Max_used_connections", status, errs, "max_used_connections"),
		"open": common.MapStr{
			"files":   h.ToInt("Open_files", status, errs, "open.files"),
			"streams": h.ToInt("Open_streams", status, errs, "open.streams"),
			"tables":  h.ToInt("Open_tables", status, errs, "open.tables"),
		},
		"opened_tables": h.ToInt("Opened_tables", status, errs, "opened_tables"),
	}
	h.RemoveErroredKeys(event, errs)

	return event
}
