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

package metrics

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
	schema = s.Schema{
		"rule": c.Dict("rule", s.Schema{
			"name":          					c.Str("name", s.Optional),
			"id":   									c.Str("id", s.Optional),
			"lastExecutionDuration":  c.Str("lastExecutionDuration", s.Optional),
			"averageDrift": 					c.Str("averageDrift", s.Optional),
			"averageDuration": 				c.Str("averageDuration", s.Optional),
			"lastExecutionTimeout": 	c.Str("lastExecutionTimeout", s.Optional),
			"totalExecutions": 				c.Str("totalExecutions", s.Optional),
		}),
		"kibana": c.Ifc("kibana"),
		"uuid":  c.Str("kibana.uuid"),
		"name":  c.Str("kibana.name"),
		"index": c.Str("kibana.index"),
		"host": s.Object{
			"name": c.Str("kibana.host"),
		},
		"transport_address":      c.Str("kibana.transport_address"),
		"version":                c.Str("kibana.version"),
		"snapshot":               c.Bool("kibana.snapshot"),
		"status":                 c.Str("kibana.status"),
		"task_manager": c.Dict("task_manager", s.Schema{
			"pending": c.Int("pending"),
		}),
	}
)

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana Metrics API response")
	}

	dataFields, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure to apply metrics schema")
	}

	event := mb.Event{ModuleFields: common.MapStr{}, RootFields: common.MapStr{}}

	// Set elasticsearch cluster id
	elasticsearchClusterID, ok := data["cluster_uuid"]
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("cluster_uuid", elastic.Kibana)
		return event.Error
	}
	event.ModuleFields.Put("elasticsearch.cluster.id", elasticsearchClusterID)

	// Set service ID
	uuid, err := dataFields.GetValue("uuid")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.id", uuid)
	dataFields.Delete("uuid")

	// Set service version
	version, err := dataFields.GetValue("version")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.version", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.version", version)
	dataFields.Delete("version")

	// Set service address
	serviceAddress, err := dataFields.GetValue("kibana.transport_address")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
		return event.Error
	}
	event.RootFields.Put("service.address", serviceAddress)

	// rule, ok := data["rule"].(map[string]interface{})
	// if !ok {
	// 	event.Error = elastic.MakeErrorForMissingField("rule", elastic.Kibana)
	// 	return event.Error
	// }
	// id, ok := rule["id"].(float64)
	// if !ok {
	// 	event.Error = elastic.MakeErrorForMissingField("rule.pid", elastic.Kibana)
	// 	return event.Error
	// }
	// event.RootFields.Put("process.pid", int(pid))

	dataFields.Delete("kibana")

	event.MetricSetFields = dataFields

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
		event.Index = index
	}

	r.Event(event)

	return nil
}
