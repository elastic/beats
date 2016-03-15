package status

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Map data to MapStr of server stats variables: http://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html
func eventMapping(status map[string]string) common.MapStr {

	event := common.MapStr{
		"aborted": common.MapStr{

			"Aborted_clients":  toInt(status["Aborted_clients"]),
			"Aborted_connects": toInt(status["Aborted_connects"]),
		},
		"binlog": common.MapStr{

			"Binlog_cache_disk_use": toInt(status["Binlog_cache_disk_use"]),
			"Binlog_cache_use":      toInt(status["Binlog_cache_use"]),
		},
		"bytes": common.MapStr{

			"Bytes_received": toInt(status["Bytes_received"]),
			"Bytes_sent":     toInt(status["Bytes_sent"]),
		},
		"Connections": toInt(status["Connections"]),
		"created": common.MapStr{

			"Created_tmp_disk_tables": toInt(status["Created_tmp_disk_tables"]),
			"Created_tmp_files":       toInt(status["Created_tmp_files"]),
			"Created_tmp_tables":      toInt(status["Created_tmp_tables"]),
		},

		"delayed": common.MapStr{
			"Delayed_errors":         toInt(status["Delayed_errors"]),
			"Delayed_insert_threads": toInt(status["Delayed_insert_threads"]),
			"Delayed_writes":         toInt(status["Delayed_writes"]),
		},
		"Flush_commands":       toInt(status["Flush_commands"]),
		"Max_used_connections": toInt(status["Max_used_connections"]),
		"open": common.MapStr{

			"Open_files":    toInt(status["Open_files"]),
			"Open_streams":  toInt(status["Open_streams"]),
			"Open_tables":   toInt(status["Open_tables"]),
			"Opened_tables": toInt(status["Opened_tables"]),
		},
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
