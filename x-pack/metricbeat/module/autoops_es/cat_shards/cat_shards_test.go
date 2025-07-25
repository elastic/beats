// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"

	"github.com/stretchr/testify/require"
)

var (
	setupSuccessfulServer = SetupSuccessfulServer()
	useNamedMetricSet     = auto_ops_testing.UseNamedMetricSet(CatShardsMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.NoError(t, data.Error)

		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}
