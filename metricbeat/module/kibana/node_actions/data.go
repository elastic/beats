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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var (
	kibanaSchema = s.Schema{
		"uuid":  c.Str("uuid"),
		"name":  c.Str("name"),
		"index": c.Str("index"),
		"host": s.Object{
			"name": c.Str("host"),
		},
		"transport_address": c.Str("transport_address"),
		"version":           c.Str("version"),
		"snapshot":          c.Bool("snapshot"),
		"status":            c.Str("status"),
	}

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
		return errors.Wrap(err, "failure parsing Kibana Node Actions API response")
	}

	event := mb.Event{ModuleFields: common.MapStr{}, RootFields: common.MapStr{}}

	// Set elasticsearch cluster id
	event.ModuleFields.Put("elasticsearch.cluster.id", data.ClusterUuid)

	kibana, _ := kibanaSchema.Apply(data.Kibana)
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana", elastic.Kibana)
		return event.Error
	}

	// Set service ID
	serviceId, err := kibana.GetValue("uuid")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.id", serviceId)

	// Set service version
	version, err := kibana.GetValue("version")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.version", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.version", version)

	// Set service address
	serviceAddress, err := kibana.GetValue("transport_address")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.address", serviceAddress)

	actionsFields, err := actionsSchema.Apply(data.Actions)
	if err != nil {
		return errors.Wrap(err, "failure to apply node actions specific schema")
	}
	event.MetricSetFields = actionsFields

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
		event.Index = index
	}

	r.Event(event)

	return nil
}
