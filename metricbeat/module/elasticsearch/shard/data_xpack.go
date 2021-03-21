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
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, cache map[string]string, content []byte) (map[string]string, error) {
	stateData := &stateStruct{}
	err := json.Unmarshal(content, stateData)
	if err != nil {
		return cache, errors.Wrap(err, "failure parsing Elasticsearch Cluster State API response")
	}

	// TODO: This is currently needed because the cluser_uuid is `na` in stateData in case not the full state is requested.
	// Will be fixed in: https://github.com/elastic/elasticsearch/pull/30656
	clusterID, err := elasticsearch.GetClusterID(m.HTTP, m.HostData().SanitizedURI+statePath, stateData.MasterNode)
	if err != nil {
		return cache, errors.Wrap(err, "failed to get cluster ID from Elasticsearch")
	}

	newCache := make(map[string]string)

	var errs multierror.Errors
	for _, index := range stateData.RoutingTable.Indices {
		for _, shards := range index.Shards {
			for _, shard := range shards {
				event, omit, err := handleShard(shard, stateData, clusterID, m.Module().Config().Period, cache, newCache)
				if err != nil {
					return cache, err
				}

				if omit {
					continue
				}

				r.Event(event)
			}
		}
	}

	return newCache, errs.Err()
}

func handleShard(shard common.MapStr, stateData *stateStruct, clusterID string, period time.Duration, cache map[string]string, newCache map[string]string) (event mb.Event, omit bool, err error) {
	fields, err := schema.Apply(shard)
	if err != nil {
		return mb.Event{}, true, errors.Wrap(err, "failure to apply shard schema")
	}

	// Handle node field: could be string or null
	err = elasticsearch.PassThruField("node", shard, fields)
	if err != nil {
		return mb.Event{}, true, errors.Wrap(err, "failure passing through node field")
	}

	// Handle relocating_node field: could be string or null
	err = elasticsearch.PassThruField("relocating_node", shard, fields)
	if err != nil {
		return mb.Event{}, true, errors.Wrap(err, "failure passing through relocating_node field")
	}

	event.RootFields = common.MapStr{
		"timestamp":    time.Now(),
		"cluster_uuid": clusterID,
		"interval_ms":  period.Nanoseconds() / 1000 / 1000,
		"type":         "shards",
		"shard":        fields,
		"state_uuid":   stateData.StateID,
	}

	// Build source_node object
	nodeID, ok := shard["node"]
	if !ok {
		return mb.Event{}, true, errors.New("a 'node' key with the node id was not found on elasticsearch response")
	}
	if nodeID != nil { // shard has not been allocated yet
		sourceNode, err := getSourceNode(nodeID.(string), stateData)
		if err != nil {
			return mb.Event{}, true, errors.Wrap(err, "failure getting source node information")
		}
		event.RootFields.Put("source_node", sourceNode)
	}

	event.ID, err = getEventID(stateData.StateID, fields)
	if err != nil {
		return mb.Event{}, true, errors.Wrap(err, "failure getting event ID")
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)

	// Create a new cache and replace old one on every fetch call
	content := fields.String()
	newCache[event.ID] = content

	old, ok := cache[event.ID]
	if !ok {
		return event, false, nil
	}

	if old != fields.String() {
		return event, false, nil
	}

	return mb.Event{}, true, nil
}

func getSourceNode(nodeID string, stateData *stateStruct) (common.MapStr, error) {
	nodeInfo, ok := stateData.Nodes[nodeID]
	if !ok {
		return nil, elastic.MakeErrorForMissingField("nodes."+nodeID, elastic.Elasticsearch)
	}

	return common.MapStr{
		"uuid": nodeID,
		"name": nodeInfo.Name,
	}, nil
}

func getEventID(stateID string, shard common.MapStr) (string, error) {
	shardID, err := getShardID(shard)
	if err != nil {
		return "", err
	}

	return stateID + ":" + shardID, nil
}

func getShardID(shard common.MapStr) (string, error) {
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
	shardNumberStr := strconv.FormatInt(shardNumberInt, 10)

	isPrimary, ok := shard["primary"].(bool)
	if !ok {
		return "", elastic.MakeErrorForMissingField("primary", elastic.Elasticsearch)
	}
	var shardType string
	if isPrimary {
		shardType = "p"
	} else {
		shardType = "r"
	}

	return nodeID + ":" + indexName + ":" + shardNumberStr + ":" + shardType, nil
}
