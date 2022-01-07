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

package task_manager

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"task_manager": c.Dict("task_manager", s.Schema{
			"pending": c.Int("pending"),
		}),
		"kibana": c.Dict("kibana", s.Schema{
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
		}),
	}
)

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana Task Manager API response")
	}

	schemaResponse, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure applying schema for Kibana Task Manager API response")
	}

	event := mb.Event{MetricSetFields: common.MapStr{}, ModuleFields: common.MapStr{}, RootFields: common.MapStr{}}

	// Set service address
	serviceAddress, err := schemaResponse.GetValue("kibana.transport_address")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
	}
	event.RootFields.Put("service.address", serviceAddress)

	// Set elasticsearch cluster id
	elasticsearchClusterID, ok := data["cluster_uuid"]
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("cluster_uuid", elastic.Kibana)
		return event.Error
	}
	event.ModuleFields.Put("elasticsearch.cluster.id", elasticsearchClusterID)

	// Set service ID
	uuid, err := schemaResponse.GetValue("kibana.uuid")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.id", uuid)

	taskManager, err := schemaResponse.GetValue("task_manager")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("task_manager", elastic.Kibana)
		return event.Error
	}
	event.MetricSetFields = taskManager.(common.MapStr)

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
		event.Index = index
	}

	r.Event(event)

	return nil
}
