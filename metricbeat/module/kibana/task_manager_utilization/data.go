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

package task_manager_utilization

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var (
	taskManagerStatsSchema = s.Schema{
		"timestamp": c.Str("timestamp"),
		"value": c.Dict("value", s.Schema{
			"load": c.Int("load"),
		}),
	}
)

type response struct {
	Stats       map[string]interface{} `json:"stats"`
	Timestamp   string                 `json:"timestamp"`
	LastUpdate  string                 `json:"last_update"`
	ProcessUuid string                 `json:"process_uuid"`
}

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failure parsing Kibana Task Manager Background Task Utilization API response: %w", err)
	}

	stats, err := taskManagerStatsSchema.Apply(data.Stats)
	if err != nil {
		return fmt.Errorf("failure to apply task_manager_utilization specific schema: %w", err)
	}

	// Set load value which is the only value we currently care about
	load, err := stats.GetValue("value.load")
	if err != nil {
		return elastic.MakeErrorForMissingField("stats.value.load", elastic.Kibana)
	}

	event := mb.Event{
		RootFields: mapstr.M{
			"service.id": data.ProcessUuid,
		},
		MetricSetFields: mapstr.M{
			"load": load,
		},
	}

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
		event.Index = index
	}

	r.Event(event)

	return nil
}
