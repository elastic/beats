// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package frozen_cache_stats

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	useNamedMetricSet = auto_ops_testing.UseNamedMetricSet(FrozenCacheStatsMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/frozen_cache_stats.*.json", auto_ops_testing.SetupSuccessfulServer(FrozenCacheStatsPath), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[CacheStatsResponse]) {
		require.NoError(t, data.Error)
		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}
