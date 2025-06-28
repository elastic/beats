// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupSuccessfulServerWithVersionedResolvedIndex = SetupSuccessfulServerWithVersionedResolvedIndex()
)

func TestSendNodeShardsEvent(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}
	info := auto_ops_testing.CreateClusterInfo("8.15.3")
	nodeToShards := []NodeShardCount{
		{
			NodeId:                    "node1",
			NodeName:                  "name1",
			Shards:                    100,
			PrimaryShards:             75,
			ReplicaShards:             25,
			InitializingShards:        2,
			InitializingPrimaryShards: 1,
			InitializingReplicaShards: 1,
			RelocatingShards:          3,
			RelocatingPrimaryShards:   1,
			RelocatingReplicaShards:   2,
		},
		{
			NodeId:                    "node2",
			NodeName:                  "name2",
			Shards:                    99,
			PrimaryShards:             25,
			ReplicaShards:             74,
			InitializingShards:        4,
			InitializingPrimaryShards: 2,
			InitializingReplicaShards: 2,
			RelocatingShards:          5,
			RelocatingPrimaryShards:   4,
			RelocatingReplicaShards:   1,
		},
	}
	transactionId := "xyz"

	sendNodeShardsEvent(reporter, &info, nodeToShards, transactionId)

	require.Equal(t, 0, len(reporter.GetErrors()))
	require.Equal(t, 1, len(reporter.GetEvents()))

	event := reporter.GetEvents()[0]

	auto_ops_testing.CheckEventWithTransactionId(t, event, info, transactionId)

	require.ElementsMatch(t, nodeToShards, auto_ops_testing.GetObjectValue(event.MetricSetFields, "node_shards_count"))
}

func TestSendNodeIndexShardsEventInBatch(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}
	info := auto_ops_testing.CreateClusterInfo("8.15.3")
	nodeIndexShards := maps.Values(getNodeIndexShards())
	transactionId := "xyz"

	sendNodeIndexShardsEvent(reporter, &info, nodeIndexShards, transactionId)

	require.Equal(t, 0, len(reporter.GetErrors()))
	require.Equal(t, 1, len(reporter.GetEvents()))

	event := reporter.GetEvents()[0]

	auto_ops_testing.CheckEventWithTransactionId(t, event, info, transactionId)

	require.ElementsMatch(t, nodeIndexShards, auto_ops_testing.GetObjectValue(event.MetricSetFields, "node_index_shards"))
}

func TestSendNodeIndexShardsEvent(t *testing.T) {
	t.Setenv(NODE_INDEX_SHARDS_PER_EVENT_NAME, "1")

	reporter := &mbtest.CapturingReporterV2{}
	info := auto_ops_testing.CreateClusterInfo("8.15.3")
	nodeIndexShards := maps.Values(getNodeIndexShards())
	transactionId := "xyz"

	sendNodeIndexShardsEvent(reporter, &info, nodeIndexShards, transactionId)

	require.Equal(t, 0, len(reporter.GetErrors()))
	require.Equal(t, len(nodeIndexShards), len(reporter.GetEvents()))

	events := reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	eventData := []NodeIndexShards{}

	for _, event := range events {
		auto_ops_testing.CheckEventWithTransactionId(t, event, info, transactionId)

		array := auto_ops_testing.GetObjectValue(event.MetricSetFields, "node_index_shards")
		eventData = append(eventData, array.([]NodeIndexShards)...)
	}

	require.ElementsMatch(t, nodeIndexShards, eventData)
}

func expectValidParsedData(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	// 2 <= len(events)
	require.LessOrEqual(t, 2, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	nodeShardsCountEvents := auto_ops_testing.GetEventsWithField(t, events, "node_shards_count")

	require.Equal(t, 1, len(nodeShardsCountEvents))
	require.Equal(t, 2, len(auto_ops_testing.GetObjectValue(nodeShardsCountEvents[0].MetricSetFields, "node_shards_count").([]NodeShardCount)))

	nodeIndexShardsEvents := auto_ops_testing.GetEventsWithField(t, events, "node_index_shards")

	require.Equal(t, 1, len(nodeIndexShardsEvents))
	require.LessOrEqual(t, 2, len(auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)))
}

func expectValidParsedDetailedShards(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
	expectValidParsedData(t, data)

	events := data.Reporter.GetEvents()

	require.Equal(t, 2, len(events))

	nodeShardCountsEvents := auto_ops_testing.GetEventsWithField(t, events, "node_shards_count")
	nodeShardCounts := auto_ops_testing.GetObjectValue(nodeShardCountsEvents[0].MetricSetFields, "node_shards_count").([]NodeShardCount)

	require.Equal(t, 1, len(nodeShardCountsEvents))
	require.Equal(t, 2, len(nodeShardCounts))

	node2 := nodeShardCounts[slices.IndexFunc(nodeShardCounts, func(node NodeShardCount) bool { return node.NodeId == "node2" })]

	require.Equal(t, "name2", node2.NodeName)
	require.Equal(t, "node2", node2.NodeId)
	require.EqualValues(t, 1, node2.Shards)
	require.EqualValues(t, 0, node2.PrimaryShards)
	require.EqualValues(t, 1, node2.ReplicaShards)
	require.EqualValues(t, 0, node2.InitializingShards)
	require.EqualValues(t, 0, node2.InitializingPrimaryShards)
	require.EqualValues(t, 0, node2.InitializingReplicaShards)
	require.EqualValues(t, 0, node2.RelocatingShards)
	require.EqualValues(t, 0, node2.RelocatingPrimaryShards)
	require.EqualValues(t, 0, node2.RelocatingReplicaShards)

	nodeIndexShardsEvents := auto_ops_testing.GetEventsWithField(t, events, "node_index_shards")
	nodeIndexShards := auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)

	require.Equal(t, 1, len(nodeIndexShardsEvents))
	require.LessOrEqual(t, 14, len(nodeIndexShards))

	myIndexNode2 := nodeIndexShards[slices.IndexFunc(nodeIndexShards, func(node NodeIndexShards) bool { return node.IndexNode == "my-index-node_id-node2" })]

	if data.Version == "7.17.0" {
		require.Equal(t, 14, len(nodeIndexShards))
		require.EqualValues(t, 14, myIndexNode2.TotalFractions)
	} else if data.Version == "8.15.3" {
		require.Equal(t, 35, len(nodeIndexShards))
		require.EqualValues(t, 35, myIndexNode2.TotalFractions)
	}

	require.Equal(t, "my-index", myIndexNode2.Index)
	require.Equal(t, "my-index-node_id-node2", myIndexNode2.IndexNode)
	require.Equal(t, "node2", myIndexNode2.NodeId)
	require.Equal(t, "name2", myIndexNode2.NodeName)
	require.Equal(t, GREEN, *myIndexNode2.IndexStatus)
	require.Equal(t, "index", *myIndexNode2.IndexType)
	require.ElementsMatch(t, []string{"alias-1"}, myIndexNode2.Aliases)
	require.Equal(t, 0, len(myIndexNode2.Attributes))
	require.Equal(t, false, *myIndexNode2.IsHidden)
	require.Equal(t, true, *myIndexNode2.IsOpen)
	require.Equal(t, false, *myIndexNode2.IsSystem)
	require.Equal(t, 1, len(myIndexNode2.AssignShards))
	require.EqualValues(t, 0, myIndexNode2.AssignShards[0].ShardNum)
	require.EqualValues(t, false, myIndexNode2.AssignShards[0].Primary)
	require.EqualValues(t, 7, *myIndexNode2.AssignShards[0].SegmentsCount)
	require.EqualValues(t, 98064, *myIndexNode2.AssignShards[0].SizeInBytes)
	require.EqualValues(t, 26, *myIndexNode2.AssignShards[0].DocsCount)
	require.Equal(t, STARTED, myIndexNode2.AssignShards[0].State)
	require.Equal(t, 0, len(myIndexNode2.InitializingShards))
	require.Equal(t, 0, len(myIndexNode2.RelocatingShards))
	require.Equal(t, 0, len(myIndexNode2.UnassignedShards))
	require.EqualValues(t, 1, myIndexNode2.Shards)
	require.EqualValues(t, 0, myIndexNode2.PrimaryShards)
	require.EqualValues(t, 1, myIndexNode2.ReplicaShards)
	require.EqualValues(t, 0, myIndexNode2.Initializing)
	require.EqualValues(t, 0, myIndexNode2.Relocating)
	require.EqualValues(t, 0, myIndexNode2.Unassigned)
	require.EqualValues(t, 0, myIndexNode2.UnassignedPrimaryShards)
	require.EqualValues(t, 0, myIndexNode2.UnassignedReplicasShards)
	require.EqualValues(t, 7, *myIndexNode2.TotalSegmentsCount)
	require.EqualValues(t, 98064, *myIndexNode2.TotalSizeInBytes)
	require.EqualValues(t, 98064, *myIndexNode2.TotalMaxShardSizeInBytes)
	require.EqualValues(t, 98064, *myIndexNode2.TotalMinShardSizeInBytes)
	require.EqualValues(t, 44, *myIndexNode2.TotalMergesTotalTime)
	require.EqualValues(t, 6, *myIndexNode2.TotalMergesTotal)
	require.EqualValues(t, 44, *myIndexNode2.TotalMergesTotalTime)
	require.EqualValues(t, 31, *myIndexNode2.GetMissingDocTotal)
	require.EqualValues(t, 1, *myIndexNode2.GetMissingDocTotalTime)
	require.EqualValues(t, 100, *myIndexNode2.SearchQueryTotal)
	require.EqualValues(t, 28, *myIndexNode2.SearchQueryTime)
	require.Nil(t, myIndexNode2.DocsCount)
	require.Nil(t, myIndexNode2.SegmentsCount)
	require.Nil(t, myIndexNode2.SizeInBytes)
	require.Nil(t, myIndexNode2.MaxShardSizeInBytes)
	require.Nil(t, myIndexNode2.MinShardSizeInBytes)
	require.Nil(t, myIndexNode2.MergesTotal)
	require.Nil(t, myIndexNode2.MergesTotalTime)
}

func expectValidParsedDetailedShardsWithCache(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
	expectValidParsedDetailedShards(t, data)

	nodeIndexShardsEvents := auto_ops_testing.GetEventsWithField(t, data.Reporter.GetEvents(), "node_index_shards")
	nodeIndexShards := auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)
	myIndexNode2 := nodeIndexShards[slices.IndexFunc(nodeIndexShards, func(node NodeIndexShards) bool { return node.IndexNode == "my-index-node_id-node2" })]

	require.NotNil(t, myIndexNode2.TimestampDiff)
	require.NotNil(t, myIndexNode2.GetMissingDocRatePerSecond)
	require.NotNil(t, myIndexNode2.SearchRatePerSecond)
	require.NotNil(t, myIndexNode2.SearchLatencyInMillis)
	// Note: We do not track replicas for write-related activity
	require.Nil(t, myIndexNode2.IndexRatePerSecond)
	require.Nil(t, myIndexNode2.IndexFailedRatePerSecond)
	require.Nil(t, myIndexNode2.MergeRatePerSecond)
	require.Nil(t, myIndexNode2.IndexLatencyInMillis)
	require.Nil(t, myIndexNode2.MergeLatencyInMillis)
}

func expectValidParsedWithoutResolvedIndexDataWithoutElasticSearchError(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
	require.ErrorContains(t, data.Error, "failed to load resolved index details")
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	// 2 <= len(events)
	require.LessOrEqual(t, 2, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	nodeShardsCountEvents := auto_ops_testing.GetEventsWithField(t, events, "node_shards_count")

	require.Equal(t, 1, len(nodeShardsCountEvents))
	require.Equal(t, 2, len(auto_ops_testing.GetObjectValue(nodeShardsCountEvents[0].MetricSetFields, "node_shards_count").([]NodeShardCount)))

	nodeIndexShardsEvents := auto_ops_testing.GetEventsWithField(t, events, "node_index_shards")

	require.Equal(t, 1, len(nodeIndexShardsEvents))
	require.LessOrEqual(t, 2, len(auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)))

	nodeIndexShards := auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)
	myIndexNode2 := nodeIndexShards[slices.IndexFunc(nodeIndexShards, func(node NodeIndexShards) bool { return node.IndexNode == "my-index-node_id-node2" })]

	require.Nil(t, myIndexNode2.IndexType)
	require.Nil(t, myIndexNode2.Aliases)
	require.Nil(t, myIndexNode2.Attributes)
	require.Nil(t, myIndexNode2.IsHidden)
	require.Nil(t, myIndexNode2.IsOpen)
	require.Nil(t, myIndexNode2.IsSystem)
}

func expectValidParsedWithoutResolvedIndexDataWithElasticSearchError(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	// 2 <= len(events)
	require.LessOrEqual(t, 2, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	nodeShardsCountEvents := auto_ops_testing.GetEventsWithField(t, events, "node_shards_count")

	require.Equal(t, 1, len(nodeShardsCountEvents))
	require.Equal(t, 2, len(auto_ops_testing.GetObjectValue(nodeShardsCountEvents[0].MetricSetFields, "node_shards_count").([]NodeShardCount)))

	nodeIndexShardsEvents := auto_ops_testing.GetEventsWithField(t, events, "node_index_shards")

	require.Equal(t, 1, len(nodeIndexShardsEvents))
	require.LessOrEqual(t, 2, len(auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)))

	nodeIndexShards := auto_ops_testing.GetObjectValue(nodeIndexShardsEvents[0].MetricSetFields, "node_index_shards").([]NodeIndexShards)
	myIndexNode2 := nodeIndexShards[slices.IndexFunc(nodeIndexShards, func(node NodeIndexShards) bool { return node.IndexNode == "my-index-node_id-node2" })]

	require.Nil(t, myIndexNode2.IndexType)
	require.Nil(t, myIndexNode2.Aliases)
	require.Nil(t, myIndexNode2.Attributes)
	require.Nil(t, myIndexNode2.IsHidden)
	require.Nil(t, myIndexNode2.IsOpen)
	require.Nil(t, myIndexNode2.IsSystem)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/cat_shards.*.json", setupSuccessfulServerWithVersionedResolvedIndex, useNamedMetricSet, expectValidParsedData, clearCache)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseWithDetails(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/cat_shards.*.json", setupSuccessfulServerWithVersionedResolvedIndex, useNamedMetricSet, expectValidParsedDetailedShards, clearCache)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseWithDetailsAndCache(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/cat_shards.*.json", setupSuccessfulServerWithVersionedResolvedIndex, useNamedMetricSet, expectValidParsedDetailedShardsWithCache, func() {
		zero := int64(0)

		nodeIndexShards := map[string]NodeIndexShards{
			"my-index-node_id-node2": {
				Index:                    "my-index",
				IndexNode:                "my-index-node_id-node2",
				NodeId:                   "node2",
				GetMissingDocTotal:       &zero,
				IndexingIndexTotal:       &zero,
				IndexingIndexTotalTime:   &zero,
				IndexingFailedIndexTotal: &zero,
				MergesTotalTime:          &zero,
				MergesTotal:              &zero,
				SearchQueryTime:          &zero,
				SearchQueryTotal:         &zero,
			},
		}

		initCache(nodeIndexShards, 10)
	})
}

func setupResolveErrorServer(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(clusterInfo)
		case CatShardsPath:
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case ResolveIndexPath:
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`Server Error`))
		default:
			t.Fatalf("Unknown request to %v", r.RequestURI)
		}
	}))
}

// Expect a valid response from Elasticsearch to create N events without index metadata
func TestProperlyHandlesInnerErrorsInResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/cat_shards.*.json", setupResolveErrorServer, useNamedMetricSet, expectValidParsedWithoutResolvedIndexDataWithoutElasticSearchError, clearCache)
}

// Expect Elasticsearch errors while creating events without index metadata
func TestProperlyHandlesInnerElasticSearchErrorsInResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/cat_shards.*.json", setupResolveElasticSearchServer, useNamedMetricSet, expectValidParsedWithoutResolvedIndexDataWithElasticSearchError, clearCache)
}
