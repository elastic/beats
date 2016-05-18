package mntr

import (
	"bufio"
	"io"
	"regexp"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	// Matches first the variable name, second the param itself
	paramMatcher = regexp.MustCompile("([^\\s]+)\\s+(.*$)")
)

func eventMapping(response io.Reader) common.MapStr {
	fullEvent := common.MapStr{}
	scanner := bufio.NewScanner(response)

	// Iterate through all events to gather data
	for scanner.Scan() {
		if match := paramMatcher.FindStringSubmatch(scanner.Text()); len(match) == 3 {
			fullEvent[match[1]] = match[2]
		} else {
			logp.Warn("Unexpected line in mntr output: %s", scanner.Text())
		}
	}

	// Manually convert and select fields which are used
	event := common.MapStr{
		"version": fullEvent["zk_version"],
		"latency": common.MapStr{
			"avg": toInt(fullEvent["zk_avg_latency"]),
			"min": toInt(fullEvent["zk_min_latency"]),
			"max": toInt(fullEvent["zk_max_latency"]),
		},
		"packets": common.MapStr{
			"received": toInt(fullEvent["zk_packets_received"]),
			"sent":     toInt(fullEvent["zk_packets_sent"]),
		},
		"num_alive_connections":      toInt(fullEvent["zk_num_alive_connections"]),
		"outstanding_requests":       toInt(fullEvent["zk_outstanding_requests"]),
		"server_state":               fullEvent["zk_server_state"],
		"znode_count":                toInt(fullEvent["zk_znode_count"]),
		"watch_count":                toInt(fullEvent["zk_watch_count"]),
		"ephemerals_count":           toInt(fullEvent["zk_ephemerals_count"]),
		"approximate_data_size":      toInt(fullEvent["zk_approximate_data_size"]),
		"open_file_descriptor_count": toInt(fullEvent["zk_open_file_descriptor_count"]),
		"max_file_descriptor_count":  toInt(fullEvent["zk_max_file_descriptor_count"]),
		"followers":                  toInt(fullEvent["zk_followers"]),
		"synced_followers":           toInt(fullEvent["zk_synced_followers"]),
		"pending_syncs":              toInt(fullEvent["zk_pending_syncs"]),
	}

	return event
}

// toInt converts value to int. In case of error, returns 0
func toInt(param interface{}) int {
	if param == nil {
		return 0
	}

	value, err := strconv.Atoi(param.(string))

	if err != nil {
		logp.Err("Error converting param to int: %s", param)
		value = 0
	}

	return value
}
