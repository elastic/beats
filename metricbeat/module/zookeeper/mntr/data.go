// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mntr

import (
	"bufio"
	"io"
	"regexp"

	"github.com/elastic/beats/v8/libbeat/common"

	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstrstr"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

var (
	// Matches first the variable name, second the param itself
	paramMatcher = regexp.MustCompile("([^\\s]+)\\s+(.*$)")
	schema       = s.Schema{
		"version": c.Str("zk_version"),
		"latency": s.Object{
			"avg": c.Float("zk_avg_latency"),
			"min": c.Float("zk_min_latency"),
			"max": c.Float("zk_max_latency"),
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
		"learners":         c.Int("zk_learners", s.Optional),
		"followers":        c.Int("zk_followers", s.Optional), // Not present anymore in ZooKeeper >= 3.6 mntr responses
		"synced_followers": c.Int("zk_synced_followers"),
		"pending_syncs":    c.Int("zk_pending_syncs"),
	}
	schemaUnix = s.Schema{
		"open_file_descriptor_count": c.Int("zk_open_file_descriptor_count"),
		"max_file_descriptor_count":  c.Int("zk_max_file_descriptor_count"),
	}
)

func eventMapping(serverId string, response io.Reader, r mb.ReporterV2, logger *logp.Logger) {
	fullEvent := map[string]interface{}{}
	scanner := bufio.NewScanner(response)

	// Iterate through all events to gather data
	for scanner.Scan() {
		if match := paramMatcher.FindStringSubmatch(scanner.Text()); len(match) == 3 {
			fullEvent[match[1]] = match[2]
		} else {
			logger.Infof("Unexpected line in mntr output: %s", scanner.Text())
		}
	}

	event, _ := schema.Apply(fullEvent)
	e := mb.Event{RootFields: common.MapStr{}}
	e.RootFields.Put("service.node.name", serverId)

	if version, ok := event["version"]; ok {
		e.RootFields.Put("service.version", version)
		delete(event, "version")
	}

	_, hasFollowers := fullEvent["zk_followers"]
	_, hasLearners := fullEvent["zk_learners"]

	// only exposed by the Leader
	if hasLearners || hasFollowers {
		schemaLeader.ApplyTo(event, fullEvent)

		// If ZK < 3.6, keep a migration to recent versions view of the "followers" field
		if followers, ok := event["followers"]; ok {
			event.Put("learners", followers)
		}

		// If ZK >= 3.6, keep a legacy view of the "learners" field
		if learners, ok := event["learners"]; ok {
			event.Put("followers", learners)
		}
	}

	// only available on Unix platforms
	if _, ok := fullEvent["zk_open_file_descriptor_count"]; ok {
		schemaUnix.ApplyTo(event, fullEvent)
	}

	e.MetricSetFields = event
	r.Event(e)
}
