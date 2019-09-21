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

package stats

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
	clusterUUID, err := m.getClusterUUID()
	if err != nil {
		return errors.Wrap(err, "could not determine cluster UUID")
	}

	// Massage info into beat
	beat := common.MapStr{
		"name":    info.Name,
		"host":    info.Hostname,
		"type":    info.Beat,
		"uuid":    info.UUID,
		"version": info.Version,
	}

	var metrics map[string]interface{}
	err = json.Unmarshal(content, &metrics)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's Stats API response")
	}

	fields := common.MapStr{
		"metrics":   metrics,
		"beat":      beat,
		"timestamp": now,
	}

	var event mb.Event
	event.RootFields = common.MapStr{
		"cluster_uuid": clusterUUID,
		"timestamp":    now,
		"interval_ms":  m.calculateIntervalMs(),
		"type":         "beats_stats",
		"beats_stats":  fields,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Beats)

	r.Event(event)
	return nil
}

func (m *MetricSet) calculateIntervalMs() int64 {
	return m.Module().Config().Period.Nanoseconds() / 1000 / 1000
}

func (m *MetricSet) getClusterUUID() (string, error) {
	state, err := beat.GetState(m.MetricSet)
	if err != nil {
		return "", errors.Wrap(err, "could not get state information")
	}

	clusterUUID := state.Monitoring.ClusterUUID
	if clusterUUID != "" {
		return clusterUUID, nil
	}

	if state.Output.Name != "elasticsearch" {
		return "", nil
	}

	clusterUUID = state.Outputs.Elasticsearch.ClusterUUID
	if clusterUUID == "" {
		// Output is ES but cluster UUID could not be determined. No point sending monitoring
		// data with empty cluster UUID since it will not be associated with the correct ES
		// production cluster. Log error instead.
		return "", beat.ErrClusterUUID
	}

	return clusterUUID, nil

}
