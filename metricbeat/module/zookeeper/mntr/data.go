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
	schema       = h.NewSchema(common.MapStr{
		"version": h.Str("zk_version"),
		"latency": common.MapStr{
			"avg": h.Int("zk_avg_latency"),
			"min": h.Int("zk_min_latency"),
			"max": h.Int("zk_max_latency"),
		},
		"packets": common.MapStr{
			"received": h.Int("zk_packets_received"),
			"sent":     h.Int("zk_packets_sent"),
		},
		"num_alive_connections": h.Int("zk_num_alive_connections"),
		"outstanding_requests":  h.Int("zk_outstanding_requests"),
		"server_state":          h.Str("zk_server_state"),
		"znode_count":           h.Int("zk_znode_count"),
		"watch_count":           h.Int("zk_watch_count"),
		"ephemerals_count":      h.Int("zk_ephemerals_count"),
		"approximate_data_size": h.Int("zk_approximate_data_size"),
	})
	schemaLeader = h.NewSchema(common.MapStr{
		"followers":        h.Int("zk_followers"),
		"synced_followers": h.Int("zk_synced_followers"),
		"pending_syncs":    h.Int("zk_pending_syncs"),
	})
	schemaUnix = h.NewSchema(common.MapStr{
		"open_file_descriptor_count": h.Int("zk_open_file_descriptor_count"),
		"max_file_descriptor_count":  h.Int("zk_max_file_descriptor_count"),
	})
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

	event := schema.Apply(fullEvent)

	// only exposed by the Leader
	if _, ok := fullEvent["zk_followers"]; ok {
		schemaLeader.ApplyTo(event, fullEvent)
	}

	// only available on Unix platforms
	if _, ok := fullEvent["open_file_descriptor_count"]; ok {
		schemaUnix.ApplyTo(event, fullEvent)
	}

	return event
}
