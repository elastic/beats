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

package rule

import (
	"encoding/json"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	// "github.com/elastic/beats/v7/libbeat/logp"
)

var (
	schema = s.Schema{
		"kibana": c.Dict("kibana", kibanaSchema),
	}

	ruleSchema = s.Schema{
		"name":                  c.Str("name"),
		"id":                    c.Str("id"),
		"lastExecutionDuration": c.Int("lastExecutionDuration", s.Optional),
		"averageDrift":          c.Int("averageDrift", s.Optional),
		"averageDuration":       c.Int("averageDuration", s.Optional),
		"lastExecutionTimeout":  c.Int("lastExecutionTimeout", s.Optional),
		"totalExecutions":       c.Int("totalExecutions", s.Optional),
	}

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
)

type rulesStruct struct {
	Rules map[string]map[string]interface{} `json:"rules"`
}

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data map[string]interface{}
	ruleData := &rulesStruct{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana Rule API response")
	}

	err = json.Unmarshal(content, ruleData)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana Rule API response")
	}

	schemaResponse, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana Rule API response")
	}

	schemaResponse.Delete("rules")

	var errs multierror.Errors
	for ruleID, rule := range ruleData.Rules {
		event := mb.Event{ModuleFields: common.MapStr{}, RootFields: common.MapStr{}}

		if ruleID == "" {
			errs = append(errs, errors.Wrap(err, "no id found"))
			continue
		}

		rule, err = ruleSchema.Apply(rule)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure to apply rule schema"))
			continue
		}

		// Set service address
		serviceAddress, err := schemaResponse.GetValue("kibana.transport_address")
		if err != nil {
			errs = append(errs, elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana))
			continue
		}
		event.RootFields.Put("service.address", serviceAddress)

		// Set elasticsearch cluster id
		elasticsearchClusterID, ok := data["cluster_uuid"]
		if !ok {
			event.Error = elastic.MakeErrorForMissingField("cluster_uuid", elastic.Kibana)
			return event.Error
		}
		event.ModuleFields.Put("elasticsearch.cluster.id", elasticsearchClusterID)

		event.MetricSetFields = rule

		// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
		// When using Agent, the index name is overwritten anyways.
		if isXpack {
			index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
			event.Index = index
		}

		r.Event(event)
	}
	return errs.Err()
}
