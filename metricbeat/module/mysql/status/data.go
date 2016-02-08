package status

import (
	"github.com/elastic/beats/libbeat/common"

	"strconv"
)

// Map data to MapStr
func eventMapping(status map[string]string) common.MapStr {

	connections, _ := strconv.Atoi(status["Connections"])
	openTables, _ := strconv.Atoi(status["Open_tables"])
	openFiles, _ := strconv.Atoi(status["Open_files"])
	openStreams, _ := strconv.Atoi(status["Open_streams"])

	event := common.MapStr{
		"Connections":  connections,
		"Open_tables":  openTables,
		"Open_files":   openFiles,
		"Open_streams": openStreams,
	}

	return event
}
