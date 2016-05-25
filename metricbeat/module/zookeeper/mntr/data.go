package mntr

import (
	"bufio"
	"io"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	h "github.com/elastic/beats/metricbeat/helper"
)

var (
	// Matches first the variable name, second the param itself
	paramMatcher = regexp.MustCompile("([^\\s]+)\\s+(.*$)")
)

func eventMapping(response io.Reader) common.MapStr {
	fullEvent := map[string]string{}
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
		"version": h.ToStr("zk_version", fullEvent),
		"latency": common.MapStr{
			"avg": h.ToInt("zk_avg_latency", fullEvent),
			"min": h.ToInt("zk_min_latency", fullEvent),
			"max": h.ToInt("zk_max_latency", fullEvent),
		},
		"packets": common.MapStr{
			"received": h.ToInt("zk_packets_received", fullEvent),
			"sent":     h.ToInt("zk_packets_sent", fullEvent),
		},
		"num_alive_connections": h.ToInt("zk_num_alive_connections", fullEvent),
		"outstanding_requests":  h.ToInt("zk_outstanding_requests", fullEvent),
		"server_state":          h.ToStr("zk_server_state", fullEvent),
		"znode_count":           h.ToInt("zk_znode_count", fullEvent),
		"watch_count":           h.ToInt("zk_watch_count", fullEvent),
		"ephemerals_count":      h.ToInt("zk_ephemerals_count", fullEvent),
		"approximate_data_size": h.ToInt("zk_approximate_data_size", fullEvent),
	}

	// only exposed by the Leader
	if _, ok := fullEvent["zk_followers"]; ok {
		event["followers"] = h.ToInt("zk_followers", fullEvent)
		event["synced_followers"] = h.ToInt("zk_synced_followers", fullEvent)
		event["pending_syncs"] = h.ToInt("zk_pending_syncs", fullEvent)
	}

	// only available on Unix platforms
	if _, ok := fullEvent["open_file_descriptor_count"]; ok {
		event["open_file_descriptor_count"] = h.ToInt("zk_open_file_descriptor_count", fullEvent)
		event["max_file_descriptor_count"] = h.ToInt("zk_max_file_descriptor_count", fullEvent)
	}

	return event
}
