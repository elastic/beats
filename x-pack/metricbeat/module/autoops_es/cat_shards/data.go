// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"golang.org/x/exp/maps"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const (
	// Break apart the events into the fractions of node_index_shards documents to avoid it being rejected
	// due to size (or just being massive)
	NODE_INDEX_SHARDS_PER_EVENT_NAME string = "NODE_INDEX_SHARDS_PER_EVENT"

	UNASSIGNED   string = "UNASSIGNED"
	STARTED      string = "STARTED"
	INITIALIZING string = "INITIALIZING"
	RELOCATING   string = "RELOCATING"

	GREEN  string = "GREEN"
	YELLOW string = "YELLOW"
	RED    string = "RED"
)

type JSONShard struct {
	// fields are guaranteed to be defined
	Index            string      `json:"i"`
	ShardId          json.Number `json:"s"`
	PrimaryOrReplica string      `json:"p"`
	State            string      `json:"st"`

	// only used for assigned states
	NodeId          string      `json:"id"`
	NodeName        string      `json:"n"`
	Docs            json.Number `json:"d"`
	GetMissingTime  json.Number `json:"gmti"`
	GetMissingTotal json.Number `json:"gmto"`
	// IndexingDeleteTime  json.Number  `json:"idti"`
	// IndexingDeleteTotal json.Number  `json:"idto"`
	IndexingIndexFailed json.Number `json:"iif"`
	IndexingIndexTime   json.Number `json:"iiti"`
	IndexingIndexTotal  json.Number `json:"iito"`
	MergeTotal          json.Number `json:"mt"`
	// MergeTotalSize      json.Number `json:"mts"`
	MergeTotalTime   json.Number `json:"mtt"`
	Store            json.Number `json:"sto"`
	SegmentsCount    json.Number `json:"sc"`
	SearchQueryTime  json.Number `json:"sqti"`
	SearchQueryTotal json.Number `json:"sqto"`

	// only used for unassigned state
	UnassignedReason  *string `json:"ur"`
	UnassignedDetails *string `json:"ud"`
}

func eventsMapping(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, jsonShards *[]JSONShard) error {
	indexToShardList := make(map[string][]Shard)
	nodeShards := make(map[string]NodeShardCount)

	// deserialize JSON data to usable data grouped by index
	for _, jsonShard := range *jsonShards {
		shard := deserializeShard(jsonShard)

		// account for all shards grouped by index
		appendIndexShards(indexToShardList, jsonShard.Index, &shard)
		// account for shards on node
		appendNodeShards(nodeShards, &shard)
	}

	transactionID := utils.NewUUIDV4()

	sendNodeShardsEvent(r, info, maps.Values(nodeShards), transactionID)

	indexMetadata, err := getResolvedIndices(m)

	if err != nil {
		indexMetadata = map[string]IndexMetadata{}
		err = fmt.Errorf("failed to load resolved index details: %w", err)
		events.SendErrorEvent(err, info, r, CatShardsMetricSet, CatShardsPath, transactionID)
	}

	sendNodeIndexShardsEvent(r, info, convertToNodeIndexShards(indexToShardList, indexMetadata), transactionID)

	return err
}

func sendNodeShardsEvent(r mb.ReporterV2, info *utils.ClusterInfo, nodeToShards []NodeShardCount, transactionId string) {
	r.Event(events.CreateEvent(info, mapstr.M{"node_shards_count": nodeToShards}, transactionId))
}

func sendNodeIndexShardsEvent(r mb.ReporterV2, info *utils.ClusterInfo, nodeIndexShards []NodeIndexShards, transactionId string) {
	nodeIndexShardsPerEvent := utils.GetIntEnvParam(NODE_INDEX_SHARDS_PER_EVENT_NAME, 100)
	size := len(nodeIndexShards)

	// group node_index_shards documents into batches for efficiency
	groups := make([]mapstr.M, 0, int(math.Ceil(float64(size)/float64(nodeIndexShardsPerEvent))))

	for i := 0; i < size; i += nodeIndexShardsPerEvent {
		group := nodeIndexShards[i:min(i+nodeIndexShardsPerEvent, size)]

		groups = append(groups, mapstr.M{"node_index_shards": group})
	}

	events.CreateAndReportEvents(r, info, groups, transactionId)
}
