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

package state

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/helper/elastic"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/beat"
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, info beat.Info, content []byte) error {
	now := time.Now()

	// Massage info into beat
	beat := common.MapStr{
		"name":    info.Name,
		"host":    info.Hostname,
		"type":    info.Beat,
		"uuid":    info.UUID,
		"version": info.Version,
	}

	var state map[string]interface{}
	err := json.Unmarshal(content, &state)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's State API response")
	}

	fields := common.MapStr{
		"state":     state,
		"beat":      beat,
		"timestamp": now,
	}

	clusterUUID := getClusterUUID(state)

	var event mb.Event
	event.RootFields = common.MapStr{
		"cluster_uuid": clusterUUID,
		"timestamp":    now,
		"interval_ms":  m.calculateIntervalMs(),
		"type":         "beats_state",
		"beats_state":  fields,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Beats)

	r.Event(event)
	return nil
}

func (m *MetricSet) calculateIntervalMs() int64 {
	return m.Module().Config().Period.Nanoseconds() / 1000 / 1000
}

func getClusterUUID(state map[string]interface{}) string {
	o, exists := state["outputs"]
	if !exists {
		return ""
	}

	outputs, ok := o.(map[string]interface{})
	if !ok {
		return ""
	}

	e, exists := outputs["elasticsearch"]
	if !exists {
		return ""
	}

	elasticsearch, ok := e.(map[string]interface{})
	if !ok {
		return ""
	}

	c, exists := elasticsearch["cluster_uuid"]
	if !exists {
		return ""
	}

	clusterUUID, ok := c.(string)
	if !ok {
		return ""
	}

	return clusterUUID
}
