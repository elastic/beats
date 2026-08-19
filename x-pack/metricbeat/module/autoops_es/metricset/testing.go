// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package metricset

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/metricbeat/mb"
	mbtest "github.com/elastic/beats/v9/metricbeat/mb/testing"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

// Data passed to the FetcherCallback
type FetcherData[T any] struct {
	File      string
	Version   string
	MetricSet mb.ReportingMetricSetV2Error
	Reporter  *mbtest.CapturingReporterV2
	Server    *httptest.Server
	Config    map[string]interface{}

	// Data read from files
	ClusterInfo utils.ClusterInfo
	Data        T

	// Error that happened (if any) from running the MetricSet with the Reporter
	Error error
}

// Callback handler for tests to wrap invoking Fetch for a MetricSet
type FetcherCallback[T any] func(t *testing.T, data FetcherData[T])

// Helper function to automatically use the server's URL and just the name as the metricset.
func UseNamedMetricSet(name string) auto_ops_testing.SetupConfigCallback {
	return func(server *httptest.Server) map[string]interface{} {
		return map[string]interface{}{"module": "autoops_es",
			"metricsets": []string{name},
			"hosts":      []string{server.URL},
		}
	}
}

// Tests MetricSet with ClusterInfo fields and data. This requires a root-level _meta/test/cluster_info.*.json file with the same version as the data file.
// This will build a dynamic test for every file matched by the glob pattern
// The difference between this utility and `RunTestsForFetcherWithGlobFiles` is that this one will _not_ run `MetricSet.Fetch(Reporter)` before calling
// `fetch` and also therefore not have an `Error` set.
func RunTestsForMetricSetWithGlobFiles[T any](t *testing.T, glob string, setupServer auto_ops_testing.SetupServerCallback, setupConfig auto_ops_testing.SetupConfigCallback, fetch FetcherCallback[T]) {
	auto_ops_testing.RunTestsForGlobFiles(t, glob, func(t *testing.T, file string, version string, data []byte) {
		info, err := os.ReadFile("../_meta/test/cluster_info." + version + ".json")
		require.NoError(t, err)

		server := setupServer(t, info, data, version)
		defer server.Close()

		config := setupConfig(server)

		deserializedInfo, err := utils.DeserializeData[utils.ClusterInfo](info)
		require.NoError(t, err)

		deserializedData, err := utils.DeserializeData[T](data)
		require.NoError(t, err)

		fetch(t, FetcherData[T]{
			File:        file,
			Version:     version,
			MetricSet:   mbtest.NewReportingMetricSetV2Error(t, config),
			Reporter:    &mbtest.CapturingReporterV2{},
			Server:      server,
			Config:      config,
			ClusterInfo: *deserializedInfo,
			Data:        *deserializedData,
		})
	})
}

// Tests MetricSet's EventsMapper with ClusterInfo fields and data. This requires a root-level _meta/test/cluster_info.*.json file with the same version as the data file.
// This will build a dynamic test for every file matched by the glob pattern
// The difference between this utility and `RunTestsForFetcherWithGlobFiles` is that this one will _not_ run `MetricSet.Fetch(Reporter)` before calling
// `fetch`, but instead it will run MetricSet.Mapper and set the Error to that value.
func RunTestsForServerlessMetricSetWithGlobFiles[T any](t *testing.T, glob string, metricset string, mapper EventsMapper[T], fetch FetcherCallback[T]) {
	setupServer := auto_ops_testing.SetupSuccessfulServer("/any")
	useNamedMetricSet := UseNamedMetricSet(metricset)

	RunTestsForMetricSetWithGlobFiles(t, glob, setupServer, useNamedMetricSet, func(t *testing.T, data FetcherData[T]) {
		err := mapper(data.Reporter, &data.ClusterInfo, &data.Data)

		fetch(t, FetcherData[T]{
			File:        data.File,
			Version:     data.Version,
			MetricSet:   data.MetricSet,
			Reporter:    data.Reporter,
			Server:      data.Server,
			Config:      data.Config,
			ClusterInfo: data.ClusterInfo,
			Data:        data.Data,
			Error:       err,
		})
	})
}

// Tests Fetch with ClusterInfo fields and data. This requires a root-level _meta/test/cluster_info.*.json file with the same version as the data file.
// This will build a dynamic test for every file matched by the glob pattern
func RunTestsForFetcherWithGlobFilesAndSetup[T any](t *testing.T, glob string, setupServer auto_ops_testing.SetupServerCallback, setupConfig auto_ops_testing.SetupConfigCallback, fetch FetcherCallback[T], setup func()) {
	RunTestsForMetricSetWithGlobFiles(t, glob, setupServer, setupConfig, func(t *testing.T, data FetcherData[T]) {
		setup()

		err := data.MetricSet.Fetch(data.Reporter)

		fetch(t, FetcherData[T]{
			File:        data.File,
			Version:     data.Version,
			MetricSet:   data.MetricSet,
			Reporter:    data.Reporter,
			Server:      data.Server,
			Config:      data.Config,
			ClusterInfo: data.ClusterInfo,
			Data:        data.Data,
			Error:       err,
		})
	})
}

// Tests Fetch with ClusterInfo fields and data. This requires a root-level _meta/test/cluster_info.*.json file with the same version as the data file.
// This will build a dynamic test for every file matched by the glob pattern
func RunTestsForFetcherWithGlobFiles[T any](t *testing.T, glob string, setupServer auto_ops_testing.SetupServerCallback, setupConfig auto_ops_testing.SetupConfigCallback, fetch FetcherCallback[T]) {
	RunTestsForMetricSetWithGlobFiles(t, glob, setupServer, setupConfig, func(t *testing.T, data FetcherData[T]) {
		err := data.MetricSet.Fetch(data.Reporter)

		fetch(t, FetcherData[T]{
			File:        data.File,
			Version:     data.Version,
			MetricSet:   data.MetricSet,
			Reporter:    data.Reporter,
			Server:      data.Server,
			Config:      data.Config,
			ClusterInfo: data.ClusterInfo,
			Data:        data.Data,
			Error:       err,
		})
	})
}

func CreateClusterInfo(clusterVersion string) utils.ClusterInfo {
	return utils.ClusterInfo{
		ClusterID:   "GZbSUUMQQI-A7UcGS6vCMa",
		ClusterName: "my-cluster",
		Version: utils.ClusterInfoVersion{
			Number:       version.MustNew(clusterVersion),
			Distribution: "rpm",
		},
	}
}

// Unravel `mapstr.M.GetValue` without an error response to make it easier to assert
func GetObjectValue(obj mapstr.M, key string) interface{} {
	exists, err := obj.HasKey(key)

	if err != nil {
		return err
	} else if !exists {
		return nil
	}

	value, err := obj.GetValue(key)

	if err != nil {
		return err
	}

	return value
}
