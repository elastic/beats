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

//go:build !integration

package state

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/module/beat"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {

	files, err := filepath.Glob("./_meta/test/state.*.json")
	require.NoError(t, err)

	info := beat.Info{
		UUID: "1234",
		Beat: "helloworld",
	}

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		require.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		err = eventMapping(reporter, info, input, true)

		require.NoError(t, err, f)
		require.True(t, len(reporter.GetEvents()) >= 1, f)
		require.Equal(t, 0, len(reporter.GetErrors()), f)
	}
}

func TestUuidFromEsOutput(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}

	info := beat.Info{
		UUID: "1234",
		Beat: "testbeat",
	}

	input, err := ioutil.ReadFile("./_meta/test/uuid_es_output.json")
	require.NoError(t, err)

	err = eventMapping(reporter, info, input, true)
	require.NoError(t, err)
	require.True(t, len(reporter.GetEvents()) >= 1)
	require.Equal(t, 0, len(reporter.GetErrors()))

	event := reporter.GetEvents()[0]

	uuid, err := event.ModuleFields.GetValue("elasticsearch.cluster.id")
	require.NoError(t, err)

	require.Equal(t, "uuid_from_es_output", uuid)
}

func TestNoEventIfEsOutputButNoUuidYet(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}

	info := beat.Info{
		UUID: "1234",
		Beat: "testbeat",
	}

	input, err := ioutil.ReadFile("./_meta/test/uuid_es_output_pre_connect.json")
	require.NoError(t, err)

	err = eventMapping(reporter, info, input, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(reporter.GetEvents()))
	require.Equal(t, 0, len(reporter.GetErrors()))
}

func TestUuidFromMonitoringConfig(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}

	info := beat.Info{
		UUID: "1234",
		Beat: "testbeat",
	}

	input, err := ioutil.ReadFile("./_meta/test/uuid_monitoring_config.json")
	require.NoError(t, err)

	err = eventMapping(reporter, info, input, true)
	require.NoError(t, err)
	require.True(t, len(reporter.GetEvents()) >= 1)
	require.Equal(t, 0, len(reporter.GetErrors()))

	event := reporter.GetEvents()[0]

	uuid, err := event.ModuleFields.GetValue("elasticsearch.cluster.id")
	require.NoError(t, err)

	require.Equal(t, "uuid_from_monitoring_config", uuid)
}

func TestNoUuidInMonitoringConfig(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}

	info := beat.Info{
		UUID: "1234",
		Beat: "testbeat",
	}

	input, err := ioutil.ReadFile("./_meta/test/uuid_no_monitoring_config.json")
	require.NoError(t, err)

	err = eventMapping(reporter, info, input, true)
	require.NoError(t, err)
	require.True(t, len(reporter.GetEvents()) >= 1)
	require.Equal(t, 0, len(reporter.GetErrors()))

	event := reporter.GetEvents()[0]

	uuid, err := event.ModuleFields.GetValue("elasticsearch.cluster.id")
	require.NoError(t, err)

	require.Equal(t, "", uuid)
}
