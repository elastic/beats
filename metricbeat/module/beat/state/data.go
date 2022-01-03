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

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/beat"
)

var (
	schema = s.Schema{
		"management": c.Dict("management", s.Schema{
			"enabled": c.Bool("enabled"),
		}),
		"service": c.Dict("service", s.Schema{
			"id":      c.Str("id"),
			"name":    c.Str("name"),
			"version": c.Str("version"),
		}),
		"module": c.Dict("module", s.Schema{
			"count": c.Int("count"),
		}),
		"output": c.Dict("output", s.Schema{
			"name": c.Str("name"),
		}),
		"queue": c.Dict("queue", s.Schema{
			"name": c.Str("name"),
		}),
		"host": c.Dict("host", s.Schema{
			"architecture":  c.Str("architecture"),
			"containerized": c.Str("containerized"),
			"hostname":      c.Str("hostname"),
			"id":            c.Str("id"),
			"os": c.Dict("os", s.Schema{
				"family":   c.Str("architecture"),
				"kernel":   c.Str("kernel"),
				"name":     c.Str("name"),
				"platform": c.Str("platform"),
				"version":  c.Str("version"),
			}),
		}),
	}
)

func eventMapping(r mb.ReporterV2, info beat.Info, content []byte, isXpack bool) error {
	event := mb.Event{
		RootFields:   common.MapStr{},
		ModuleFields: common.MapStr{},
	}

	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's State API response")
	}

	event.MetricSetFields, _ = schema.Apply(data)

	clusterUUID := getMonitoringClusterUUID(data)
	if clusterUUID == "" {
		if isOutputES(data) {
			clusterUUID = getClusterUUID(data)
			if clusterUUID != "" {
				event.ModuleFields.Put("elasticsearch.cluster.id", clusterUUID)

				if event.MetricSetFields != nil {
					event.MetricSetFields.Put("cluster.uuid", clusterUUID)
				}
			}
		}
	}

	event.MetricSetFields, _ = schema.Apply(data)

	if event.MetricSetFields != nil {
		event.MetricSetFields.Put("cluster.uuid", clusterUUID)
		event.MetricSetFields.Put("beat", common.MapStr{
			"name":    info.Name,
			"host":    info.Hostname,
			"type":    info.Beat,
			"uuid":    info.UUID,
			"version": info.Version,
		})
	}

	//Extract ECS fields from the host key
	host, ok := event.MetricSetFields["host"]
	if ok {
		hostMap, ok := host.(common.MapStr)
		if ok {
			arch, ok := hostMap["architecture"]
			if ok {
				event.RootFields.Put("host.architecture", arch)
				delete(hostMap, "architecture")
			}

			hostname, ok := hostMap["hostname"]
			if ok {
				event.RootFields.Put("host.hostname", hostname)
				delete(hostMap, "hostname")
			}

			id, ok := hostMap["id"]
			if ok {
				event.RootFields.Put("host.id", id)
				delete(hostMap, "id")
			}

			name, ok := hostMap["name"]
			if ok {
				event.RootFields.Put("host.name", name)
				delete(hostMap, "name")
			}
		}
		event.MetricSetFields["host"] = hostMap
	}

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Beats)
		event.Index = index
	}

	r.Event(event)

	return nil
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

func isOutputES(state map[string]interface{}) bool {
	o, exists := state["output"]
	if !exists {
		return false
	}

	output, ok := o.(map[string]interface{})
	if !ok {
		return false
	}

	n, exists := output["name"]
	if !exists {
		return false
	}

	name, ok := n.(string)
	if !ok {
		return false
	}

	return name == "elasticsearch"
}

func getMonitoringClusterUUID(state map[string]interface{}) string {
	m, exists := state["monitoring"]
	if !exists {
		return ""
	}

	monitoring, ok := m.(map[string]interface{})
	if !ok {
		return ""
	}

	c, exists := monitoring["cluster_uuid"]
	if !exists {
		return ""
	}

	clusterUUID, ok := c.(string)
	if !ok {
		return ""
	}

	return clusterUUID
}
