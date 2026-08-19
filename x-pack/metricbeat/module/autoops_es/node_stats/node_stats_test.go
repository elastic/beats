// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package node_stats

import (
	"testing"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	useNamedMetricSet = auto_ops_testing.UseNamedMetricSet(NodesStatsMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesStats]) {
		require.NoError(t, data.Error)

		require.LessOrEqual(t, 2, len(data.Reporter.GetEvents()))
	})
}
