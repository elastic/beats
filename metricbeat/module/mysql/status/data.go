package status

import (
	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

// Map data to MapStr of server stats variables: http://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html
// This is only a subset of the available values
func eventMapping(status map[string]string) common.MapStr {

	event := common.MapStr{
		"aborted": common.MapStr{
			"clients":  h.ToInt("Aborted_clients", status),
			"connects": h.ToInt("Aborted_connects", status),
		},
		"binlog": common.MapStr{
			"cache": common.MapStr{
				"disk_use": h.ToInt("Binlog_cache_disk_use", status),
				"use":      h.ToInt("Binlog_cache_use", status),
			},
		},
		"bytes": common.MapStr{
			"received": h.ToInt("Bytes_received", status),
			"sent":     h.ToInt("Bytes_sent", status),
		},
		"connections": h.ToInt("Connections", status),
		"created": common.MapStr{
			"tmp": common.MapStr{
				"disk_tables": h.ToInt("Created_tmp_disk_tables", status),
				"files":       h.ToInt("Created_tmp_files", status),
				"tables":      h.ToInt("Created_tmp_tables", status),
			},
		},
		"delayed": common.MapStr{
			"errors":         h.ToInt("Delayed_errors", status),
			"insert_threads": h.ToInt("Delayed_insert_threads", status),
			"writes":         h.ToInt("Delayed_writes", status),
		},
		"flush_commands":       h.ToInt("Flush_commands", status),
		"max_used_connections": h.ToInt("Max_used_connections", status),
		"open": common.MapStr{
			"files":   h.ToInt("Open_files", status),
			"streams": h.ToInt("Open_streams", status),
			"tables":  h.ToInt("Open_tables", status),
		},
		"opened_tables": h.ToInt("Opened_tables", status),
	}

	return event
}
