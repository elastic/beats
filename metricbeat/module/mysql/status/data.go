package status

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Map data to MapStr of server stats variables: http://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html
// This is only a subset of the available values
func eventMapping(status map[string]string) common.MapStr {

	event := common.MapStr{
		"aborted": common.MapStr{
			"clients":  toInt(status["Aborted_clients"]),
			"connects": toInt(status["Aborted_connects"]),
		},
		"binlog": common.MapStr{
			"cache": common.MapStr{
				"disk_use": toInt(status["Binlog_cache_disk_use"]),
				"use":      toInt(status["Binlog_cache_use"]),
			},
		},
		"bytes": common.MapStr{
			"received": toInt(status["Bytes_received"]),
			"sent":     toInt(status["Bytes_sent"]),
		},
		"connections": toInt(status["Connections"]),
		"created": common.MapStr{
			"tmp": common.MapStr{
				"disk_tables": toInt(status["Created_tmp_disk_tables"]),
				"files":       toInt(status["Created_tmp_files"]),
				"tables":      toInt(status["Created_tmp_tables"]),
			},
		},
		"delayed": common.MapStr{
			"errors":         toInt(status["Delayed_errors"]),
			"insert_threads": toInt(status["Delayed_insert_threads"]),
			"writes":         toInt(status["Delayed_writes"]),
		},
		"flush_commands":       toInt(status["Flush_commands"]),
		"max_used_connections": toInt(status["Max_used_connections"]),
		"open": common.MapStr{
			"files":   toInt(status["Open_files"]),
			"streams": toInt(status["Open_streams"]),
			"tables":  toInt(status["Open_tables"]),
		},
		"opened_tables": toInt(status["Opened_tables"]),
	}

	return event
}

func toInt(param string) int {
	value, err := strconv.Atoi(param)

	if err != nil {
		logp.Err("Error converting param to int: %s", param)
		value = 0
	}

	return value
}
