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

package shard

import (
	"encoding/json"
	"strconv"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"fmt"

	"github.com/joeshaw/multierror"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"state":   c.Str("state"),
		"primary": c.Bool("primary"),
		"index":   c.Str("index"),
		"shard":   c.Int("shard"),
	}
)

type stateStruct struct {
	ClusterID   string `json:"cluster_uuid"`
	ClusterName string `json:"cluster_name"`
	StateID     string `json:"state_uuid"`
	MasterNode  string `json:"master_node"`
	Nodes       map[string]struct {
		Name string `json:"name"`
	} `json:"nodes"`
	RoutingTable struct {
		Indices map[string]struct {
			Shards map[string][]map[string]interface{} `json:"shards"`
		} `json:"indices"`
	} `json:"routing_table"`
}

func eventsMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	stateData := &stateStruct{}
	err := json.Unmarshal(content, stateData)
	if err != nil {
		return fmt.Errorf("failure parsing Elasticsearch Cluster State API response: %w", err)
	}

	var errs multierror.Errors
	for _, index := range stateData.RoutingTable.Indices {
		for _, shards := range index.Shards {
			for i, shard := range shards {
				event := mb.Event{
					ModuleFields: mapstr.M{},
				}

				_, _ = event.ModuleFields.Put("cluster.state.id", stateData.StateID)
				_, _ = event.ModuleFields.Put("cluster.stats.state.state_uuid", stateData.StateID)
				_, _ = event.ModuleFields.Put("cluster.id", stateData.ClusterID)
				_, _ = event.ModuleFields.Put("cluster.name", stateData.ClusterName)

				fields, err := schema.Apply(shard)
				if err != nil {
					errs = append(errs, fmt.Errorf("failure applying shard schema: %w", err))
					continue
				}

				// Handle node field: could be string or null
				err = elasticsearch.PassThruField("node", shard, fields)
				if err != nil {
					errs = append(errs, fmt.Errorf("failure passing through node field: %w", err))
					continue
				}

				// Handle relocating_node field: could be string or null
				err = elasticsearch.PassThruField("relocating_node", shard, fields)
				if err != nil {
					errs = append(errs, fmt.Errorf("failure passing through relocating_node field: %w", err))
					continue
				}

				event.ID, err = generateHashForEvent(stateData.StateID, fields, i)
				if err != nil {
					errs = append(errs, fmt.Errorf("failure getting event ID: %w", err))
					continue
				}

				event.MetricSetFields = fields

				nodeID, ok := shard["node"]
				if !ok {
					continue
				}
				if nodeID != nil { // shard has not been allocated yet
					_, _ = event.ModuleFields.Put("node.id", nodeID)
					delete(fields, "node")

					sourceNode, err := getSourceNode(nodeID.(string), stateData)
					if err != nil {
						errs = append(errs, fmt.Errorf("failure getting source node information: %w", err))
						continue
					}
					_, _ = event.ModuleFields.Put("node.name", sourceNode["name"])
					_, _ = event.MetricSetFields.Put("source_node", sourceNode)
				}

				_, _ = event.ModuleFields.Put("index.name", fields["index"])
				delete(fields, "index")

				_, _ = event.MetricSetFields.Put("number", fields["shard"])
				delete(event.MetricSetFields, "shard")

				delete(event.MetricSetFields, "relocating_node")
				relocatingNode := fields["relocating_node"]
				_, _ = event.MetricSetFields.Put("relocating_node.name", relocatingNode)
				_, _ = event.MetricSetFields.Put("relocating_node.id", relocatingNode)

				// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
				// When using Agent, the index name is overwritten anyways.
				if isXpack {
					index := elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
					event.Index = index
				}

				r.Event(event)
			}
		}
	}

	return errs.Err()
}

func getSourceNode(nodeID string, stateData *stateStruct) (mapstr.M, error) {
	nodeInfo, ok := stateData.Nodes[nodeID]
	if !ok {
		return nil, elastic.MakeErrorForMissingField("nodes."+nodeID, elastic.Elasticsearch)
	}

	return mapstr.M{
		"uuid": nodeID,
		"name": nodeInfo.Name,
	}, nil
}

// Note: This function may generate duplicate IDs, but those will be dropped since libbeat
// ignores the 409 status code
// https://github.com/elastic/beats/blob/main/libbeat/outputs/elasticsearch/client.go#L396
func generateHashForEvent(stateID string, shard mapstr.M, index int) (string, error) {
	var nodeID string
	if shard["node"] == nil {
		nodeID = "_na"
	} else {
		var ok bool
		nodeID, ok = shard["node"].(string)
		if !ok {
			return "", elastic.MakeErrorForMissingField("node", elastic.Elasticsearch)
		}
	}

	indexName, ok := shard["index"].(string)
	if !ok {
		return "", elastic.MakeErrorForMissingField("index", elastic.Elasticsearch)
	}

	shardNumberInt, ok := shard["shard"].(int64)
	if !ok {
		return "", elastic.MakeErrorForMissingField("shard", elastic.Elasticsearch)
	}
	shardNumberStr := "s" + strconv.FormatInt(shardNumberInt, 10)

	isPrimary, ok := shard["primary"].(bool)
	if !ok {
		return "", elastic.MakeErrorForMissingField("primary", elastic.Elasticsearch)
	}
	var shardType string
	if isPrimary {
		shardType = "p"
	} else {
		shardType = "r" + strconv.Itoa(index)
	}

	return stateID + ":" + nodeID + ":" + indexName + ":" + shardNumberStr + ":" + shardType, nil
}
