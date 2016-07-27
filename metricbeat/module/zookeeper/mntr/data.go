package mntr

import (
	"bufio"
	"io"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

var (
	// Matches first the variable name, second the param itself
	paramMatcher = regexp.MustCompile("([^\\s]+)\\s+(.*$)")
	schema_      = s.Schema{
		"version": c.Str("zk_version"),
		"latency": s.Object{
			"avg": c.Int("zk_avg_latency"),
			"min": c.Int("zk_min_latency"),
			"max": c.Int("zk_max_latency"),
		},
		"packets": s.Object{
			"received": c.Int("zk_packets_received"),
			"sent":     c.Int("zk_packets_sent"),
		},
		"num_alive_connections": c.Int("zk_num_alive_connections"),
		"outstanding_requests":  c.Int("zk_outstanding_requests"),
		"server_state":          c.Str("zk_server_state"),
		"znode_count":           c.Int("zk_znode_count"),
		"watch_count":           c.Int("zk_watch_count"),
		"ephemerals_count":      c.Int("zk_ephemerals_count"),
		"approximate_data_size": c.Int("zk_approximate_data_size"),
	}
	schemaLeader = s.Schema{
		"followers":        c.Int("zk_followers"),
		"synced_followers": c.Int("zk_synced_followers"),
		"pending_syncs":    c.Int("zk_pending_syncs"),
	}
	schemaUnix = s.Schema{
		"open_file_descriptor_count": c.Int("zk_open_file_descriptor_count"),
		"max_file_descriptor_count":  c.Int("zk_max_file_descriptor_count"),
	}
)

func eventMapping(response io.Reader) common.MapStr {
	fullEvent := map[string]interface{}{}
	scanner := bufio.NewScanner(response)

	// Iterate through all events to gather data
	for scanner.Scan() {
		if match := paramMatcher.FindStringSubmatch(scanner.Text()); len(match) == 3 {
			fullEvent[match[1]] = match[2]
		} else {
			logp.Warn("Unexpected line in mntr output: %s", scanner.Text())
		}
	}

	event := schema_.Apply(fullEvent)

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
