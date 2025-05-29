// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

func TestGetResolvedIndicesError(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupResolveErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.ErrorContains(t, data.Error, "failed to load resolved index details")
	})
}

func TestGetResolvedIndicesReturnsResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.NoError(t, data.Error)
	})
}

func TestParseResolvedIndicesResponseReturnsEmpty(t *testing.T) {
	emptyResponse := ResolvedApiResponse{Indices: []ResolvedIndices{}}

	require.Equal(t, 0, len(parseResolvedIndicesResponse(&emptyResponse)))
}

func TestParseResolvedIndicesResponse(t *testing.T) {
	response := ResolvedApiResponse{Indices: []ResolvedIndices{
		{Name: "my-index-1"},
		{Name: "my-index-2", Attributes: []string{"open"}},
		{Name: "my-index-3", Attributes: []string{"open", "system"}},
		{Name: "my-index-4", Attributes: []string{"hidden", "open", "system"}},
		{Name: "my-index-5", Attributes: []string{"hidden", "open"}},
		{Name: "my-index-6", DataStreams: "my-data-stream-1", Attributes: []string{"open"}},
		{Name: "my-index-7", DataStreams: []string{"my-data-stream-2"}, Attributes: []string{"open"}},
		{Name: "my-index-8", DataStreams: []string{"my-data-stream-3"}, Aliases: "alias-1", Attributes: []string{"open"}},
		{Name: "my-index-9", DataStreams: []string{"my-data-stream-4"}, Aliases: []string{"alias-1", "alias-2"}, Attributes: []string{"open"}},
		{Name: "my-index-10", Aliases: []string{"alias-1", "alias-2"}, Attributes: []string{"open"}},
		{Name: "my-index-11", Attributes: []string{"xyz"}},
		{Name: "my-index-12", Attributes: []string{"abc", "open"}},
	}}

	indexMetadata := parseResolvedIndicesResponse(&response)

	require.Equal(t, len(response.Indices), len(indexMetadata))

	// my-index-1
	require.Equal(t, "index", indexMetadata["my-index-1"].indexType)
	require.Nil(t, indexMetadata["my-index-1"].aliases)
	require.Nil(t, indexMetadata["my-index-1"].attributes)
	require.False(t, indexMetadata["my-index-1"].open)
	require.False(t, indexMetadata["my-index-1"].system)
	require.False(t, indexMetadata["my-index-1"].hidden)

	// my-index-2
	require.Equal(t, "index", indexMetadata["my-index-2"].indexType)
	require.Nil(t, indexMetadata["my-index-2"].aliases)
	require.Nil(t, indexMetadata["my-index-2"].attributes)
	require.True(t, indexMetadata["my-index-2"].open)
	require.False(t, indexMetadata["my-index-2"].system)
	require.False(t, indexMetadata["my-index-2"].hidden)

	// my-index-3
	require.Equal(t, "index", indexMetadata["my-index-3"].indexType)
	require.Nil(t, indexMetadata["my-index-3"].aliases)
	require.Nil(t, indexMetadata["my-index-3"].attributes)
	require.True(t, indexMetadata["my-index-3"].open)
	require.True(t, indexMetadata["my-index-3"].system)
	require.False(t, indexMetadata["my-index-3"].hidden)

	// my-index-4
	require.Equal(t, "index", indexMetadata["my-index-4"].indexType)
	require.Nil(t, indexMetadata["my-index-4"].aliases)
	require.Nil(t, indexMetadata["my-index-4"].attributes)
	require.True(t, indexMetadata["my-index-4"].open)
	require.True(t, indexMetadata["my-index-4"].system)
	require.True(t, indexMetadata["my-index-4"].hidden)

	// my-index-5
	require.Equal(t, "index", indexMetadata["my-index-5"].indexType)
	require.Nil(t, indexMetadata["my-index-5"].aliases)
	require.Nil(t, indexMetadata["my-index-5"].attributes)
	require.True(t, indexMetadata["my-index-5"].open)
	require.False(t, indexMetadata["my-index-5"].system)
	require.True(t, indexMetadata["my-index-5"].hidden)

	// my-index-6
	require.Equal(t, "data_stream", indexMetadata["my-index-6"].indexType)
	require.ElementsMatch(t, []string{"my-data-stream-1"}, indexMetadata["my-index-6"].aliases)
	require.Nil(t, indexMetadata["my-index-6"].attributes)
	require.True(t, indexMetadata["my-index-6"].open)
	require.False(t, indexMetadata["my-index-6"].system)
	require.False(t, indexMetadata["my-index-6"].hidden)

	// my-index-7
	require.Equal(t, "data_stream", indexMetadata["my-index-7"].indexType)
	require.ElementsMatch(t, []string{"my-data-stream-2"}, indexMetadata["my-index-7"].aliases)
	require.Nil(t, indexMetadata["my-index-7"].attributes)
	require.True(t, indexMetadata["my-index-7"].open)
	require.False(t, indexMetadata["my-index-7"].system)
	require.False(t, indexMetadata["my-index-7"].hidden)

	// my-index-8
	require.Equal(t, "data_stream", indexMetadata["my-index-8"].indexType)
	require.ElementsMatch(t, []string{"my-data-stream-3", "alias-1"}, indexMetadata["my-index-8"].aliases)
	require.Nil(t, indexMetadata["my-index-8"].attributes)
	require.True(t, indexMetadata["my-index-8"].open)
	require.False(t, indexMetadata["my-index-8"].system)
	require.False(t, indexMetadata["my-index-8"].hidden)

	// my-index-9
	require.Equal(t, "data_stream", indexMetadata["my-index-9"].indexType)
	require.ElementsMatch(t, []string{"my-data-stream-4", "alias-1", "alias-2"}, indexMetadata["my-index-9"].aliases)
	require.Nil(t, indexMetadata["my-index-9"].attributes)
	require.True(t, indexMetadata["my-index-9"].open)
	require.False(t, indexMetadata["my-index-9"].system)
	require.False(t, indexMetadata["my-index-9"].hidden)

	// my-index-10
	require.Equal(t, "index", indexMetadata["my-index-10"].indexType)
	require.ElementsMatch(t, []string{"alias-1", "alias-2"}, indexMetadata["my-index-10"].aliases)
	require.Nil(t, indexMetadata["my-index-10"].attributes)
	require.True(t, indexMetadata["my-index-10"].open)
	require.False(t, indexMetadata["my-index-10"].system)
	require.False(t, indexMetadata["my-index-10"].hidden)

	// my-index-11
	require.Equal(t, "index", indexMetadata["my-index-11"].indexType)
	require.Nil(t, indexMetadata["my-index-11"].aliases)
	require.ElementsMatch(t, []string{"xyz"}, indexMetadata["my-index-11"].attributes)
	require.False(t, indexMetadata["my-index-11"].open)
	require.False(t, indexMetadata["my-index-11"].system)
	require.False(t, indexMetadata["my-index-11"].hidden)

	// my-index-12
	require.Equal(t, "index", indexMetadata["my-index-12"].indexType)
	require.Nil(t, indexMetadata["my-index-12"].aliases)
	require.ElementsMatch(t, []string{"abc"}, indexMetadata["my-index-12"].attributes)
	require.True(t, indexMetadata["my-index-12"].open)
	require.False(t, indexMetadata["my-index-12"].system)
	require.False(t, indexMetadata["my-index-12"].hidden)
}
