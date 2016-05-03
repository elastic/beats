/*

mntr produces the following output which is parsed:

	zk_version      3.5.1-alpha-1693007, built on 07/28/2015 07:19 GMT
	zk_avg_latency  0
	zk_max_latency  1789
	zk_min_latency  0
	zk_packets_received     22152032
	zk_packets_sent 30959914
	zk_num_alive_connections        1033
	zk_outstanding_requests 0
	zk_server_state leader
	zk_znode_count  242609
	zk_watch_count  940522
	zk_ephemerals_count     8565
	zk_approximate_data_size        372143564
	zk_open_file_descriptor_count   1083
	zk_max_file_descriptor_count    1048576
	zk_followers    5
	zk_synced_followers     2
	zk_pending_syncs        0


*/
package mntr

import (
	"bufio"
	"fmt"
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
			fmt.Printf("Event: %+v\n", scanner.Text())
		} else {
			logp.Warn("Unexpected line in mntr output: %s", scanner.Text())
		}
	}

	// Manually convert and select fields which are used
	event := common.MapStr{
		"zk_version":                    fullEvent["zk_version"],
		"zk_avg_latency":                toInt(fullEvent["zk_avg_latency"]),
		"zk_min_latency":                toInt(fullEvent["zk_min_latency"]),
		"zk_max_latency":                toInt(fullEvent["zk_max_latency"]),
		"zk_packets_received":           toInt(fullEvent["zk_packets_received"]),
		"zk_packets_sent":               toInt(fullEvent["zk_packets_sent"]),
		"zk_num_alive_connections":      toInt(fullEvent["zk_num_alive_connections"]),
		"zk_outstanding_requests":       toInt(fullEvent["zk_outstanding_requests"]),
		"zk_server_state":               fullEvent["zk_server_state"],
		"zk_znode_count":                toInt(fullEvent["zk_znode_count"]),
		"zk_watch_count":                toInt(fullEvent["zk_watch_count"]),
		"zk_ephemerals_count":           toInt(fullEvent["zk_ephemerals_count"]),
		"zk_approximate_data_size":      toInt(fullEvent["zk_approximate_data_size"]),
		"zk_open_file_descriptor_count": toInt(fullEvent["zk_open_file_descriptor_count"]),
		"zk_max_file_descriptor_count":  toInt(fullEvent["zk_max_file_descriptor_count"]),
		"zk_followers":                  toInt(fullEvent["zk_followers"]),
		"zk_synced_followers":           toInt(fullEvent["zk_synced_followers"]),
		"zk_pending_syncs":              toInt(fullEvent["zk_pending_syncs"]),
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
