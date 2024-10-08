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

package node_actions

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/module/kibana"
	"github.com/elastic/elastic-agent-libs/mapstr"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var (
	actionsSchema = s.Schema{
		"failures":   c.Int("failures"),
		"executions": c.Int("executions"),
		"timeouts":   c.Int("timeouts"),
	}
)

type response struct {
	Actions     map[string]interface{} `json:"node_actions"`
	Kibana      map[string]interface{} `json:"kibana"`
	ClusterUuid string                 `json:"cluster_uuid"`
}

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failure parsing Kibana Node Actions API response: %w", err)
	}

	kibanaData, err := kibana.KibanaSchema.Apply(data.Kibana)
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana", elastic.Kibana)
	}

	// Set service ID
	serviceId, err := kibanaData.GetValue("uuid")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
	}

	// Set service version
	version, err := kibanaData.GetValue("version")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.version", elastic.Kibana)
	}

	// Set service address
	serviceAddress, err := kibanaData.GetValue("transport_address")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
	}

	actionsFields, err := actionsSchema.Apply(data.Actions)
	if err != nil {
		return fmt.Errorf("failure to apply node actions specific schema: %w", err)
	}

	event := mb.Event{
		ModuleFields: mapstr.M{
			"elasticsearch.cluster.id": data.ClusterUuid,
		},
		RootFields: mapstr.M{
			"service.id":      serviceId,
			"service.version": version,
		},
		MetricSetFields: actionsFields,
		Host:            fmt.Sprintf("%v", serviceAddress),
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
