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

package cluster_rules

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

	rulesSchema = s.Schema{
		"overdue": c.Dict("overdue", s.Schema{
			"count": c.Int("count"),
			"delay": c.Dict("delay", s.Schema{
				"p50": c.Float("p50"),
				"p99": c.Float("p99"),
			}),
		}),
	}
)

type response struct {
	Rules       map[string]interface{} `json:"cluster_rules"`
	Kibana      map[string]interface{} `json:"kibana"`
	ClusterUuid string                 `json:"cluster_uuid"`
}

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data response
	err := json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failure parsing Kibana Cluster Rules API response: %w", err)
	}

	rootFields := mapstr.M{}
	moduleFields := mapstr.M{}

	// Set elasticsearch cluster id
	moduleFields.Put("elasticsearch.cluster.id", data.ClusterUuid)

	kibana, _ := kibanaSchema.Apply(data.Kibana)
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana", elastic.Kibana)
	}

	// Set service ID
	serviceId, err := kibana.GetValue("uuid")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
	}
	rootFields.Put("service.id", serviceId)

	// Set service version
	version, err := kibana.GetValue("version")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.version", elastic.Kibana)
	}
	rootFields.Put("service.version", version)

	// Set service address
	serviceAddress, err := kibana.GetValue("transport_address")
	if err != nil {
		return elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
	}
	rootFields.Put("service.address", serviceAddress)

	rulesFields, err := rulesSchema.Apply(data.Rules)
	if err != nil {
		return fmt.Errorf("failure to apply cluster rules specific schema: %w", err)
	}

	event := mb.Event{ModuleFields: moduleFields, RootFields: rootFields, MetricSetFields: rulesFields}

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
		event.Index = index
	}

	r.Event(event)

	return nil
}
