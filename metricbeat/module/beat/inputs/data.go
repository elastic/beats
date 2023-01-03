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

package inputs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/beat"
)

var (
	schema = s.Schema{
		"input": c.Dict("beat.input", s.Schema{
			"id":   c.Str("id"),
			"type": c.Str("type"),
			"metrics": c.Dict("metrics", s.Schema{
				"metric1_int":   c.Int("metric1_int"),
				"metric2_float": c.Float("metric2_float"),
			}),
		}),
	}
)

func deriveSchema(inputEvent map[string]interface{}) s.Schema {
	var schema_metrics = make(s.Schema)

	if v1, ok := inputEvent["beat"]; ok {
		v11 := v1.(map[string]interface{})
		if v2, ok := v11["input"]; ok {
			v21 := v2.(map[string]interface{})
			if v3, ok := v21["metrics"]; ok {
				v31 := v3.(map[string]interface{})

				for k, v := range v31 {
					fmt.Println(k, "=>", v)
					// switch{
					// 	case strings.Contains(k, "int"):

					// }
					if strings.Contains(k, "int") {
						schema_metrics[k] = c.Int(k)
					} else if strings.Contains(k, "float") {
						schema_metrics[k] = c.Float(k)
					}
				}
			}
		}
	}
	derivedSchema := s.Schema{
		"input": c.Dict("beat.input", s.Schema{
			"id":      c.Str("id"),
			"type":    c.Str("type"),
			"metrics": c.Dict("metrics", schema_metrics),
		}),
	}
	return derivedSchema
}

func eventMapping(r mb.ReporterV2, info beat.Info, clusterUUID string, content []byte, isXpack bool) error {
	fmt.Println("printing the schema variable provided: ", schema)
	var data []map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's Inputs API response")
	}

	for _, inputEvent := range data {
		derivedSchema := deriveSchema(inputEvent)
		fmt.Println("printing the schema variable created: ", derivedSchema)
		event := mb.Event{
			RootFields:      mapstr.M{},
			ModuleFields:    mapstr.M{},
			MetricSetFields: mapstr.M{},
		}
		event.RootFields.Put("service.name", beat.ModuleName)

		event.ModuleFields.Put("id", info.UUID)
		event.ModuleFields.Put("type", info.Beat)

		if clusterUUID != "" {
			event.ModuleFields.Put("elasticsearch.cluster.id", clusterUUID)
		}

		event.MetricSetFields, _ = derivedSchema.Apply(inputEvent)
		event.MetricSetFields.Put("beat", mapstr.M{
			"name":    info.Name,
			"host":    info.Hostname,
			"type":    info.Beat,
			"uuid":    info.UUID,
			"version": info.Version,
		})

		// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
		// When using Agent, the index name is overwritten anyways.
		if isXpack {
			index := elastic.MakeXPackMonitoringIndexName(elastic.Beats)
			event.Index = index
		}

		r.Event(event)
	}
	return nil
}
