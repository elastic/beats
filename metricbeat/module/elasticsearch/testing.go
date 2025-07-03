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

package elasticsearch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/version"
)

// TestMapper tests mapping methods
func TestMapper(t *testing.T, glob string, mapper func(mb.ReporterV2, []byte) error) {
	files, err := filepath.Glob(glob)
	require.NoError(t, err)
	// Makes sure glob matches at least 1 file
	require.True(t, len(files) > 0)

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			err = mapper(reporter, input)
			require.NoError(t, err)
			require.True(t, len(reporter.GetEvents()) >= 1)
			require.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}

// TestMapperWithInfo tests mapping methods with Info fields
func TestMapperWithInfo(t *testing.T, glob string, mapper func(mb.ReporterV2, Info, []byte, bool) error) {
	files, err := filepath.Glob(glob)
	require.NoError(t, err)
	// Makes sure glob matches at least 1 file
	require.True(t, len(files) > 0, "Glob should match at least one file")

	info := Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			err = mapper(reporter, info, input, true)
			require.NoError(t, err)

			require.True(t, len(reporter.GetEvents()) >= 1)
			require.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}

func TestMapperWithExpectedEvents(
	t *testing.T,
	inputPath string,
	expectedFiles []string,
	info Info,
	isXPack bool,
	mapper func(mb.ReporterV2, Info, []byte, bool) error,
) {
	input, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = mapper(reporter, info, input, isXPack)
	require.NoError(t, err)

	events := reporter.GetEvents()

	expected := loadExpectedEventsFromFiles(t, expectedFiles)
	require.Equal(t, len(expected), len(events), "Number of events mismatch")

	for i, ev := range events {
		actualBytes, err := json.Marshal(ev)
		require.NoError(t, err)

		var actual map[string]interface{}
		err = json.Unmarshal(actualBytes, &actual)
		require.NoError(t, err)

		assert.Equal(t, expected[i], actual, fmt.Sprintf("Mismatch in event #%d", i))
	}
}

func TestMapperExpectingError(
	t *testing.T,
	inputPath string,
	info Info,
	isXPack bool,
	errorMessage string,
	mapper func(mb.ReporterV2, Info, []byte, bool) error,
) {
	input, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = mapper(reporter, info, input, isXPack)
	require.ErrorContains(t, err, errorMessage)

	events := reporter.GetEvents()
	require.Equal(t, 0, len(events), "Number of events mismatch")
}

func loadExpectedEventsFromFiles(t *testing.T, files []string) []map[string]interface{} {
	expected := make([]map[string]interface{}, 0, len(files))
	for _, f := range files {
		content, err := os.ReadFile(f)
		require.NoError(t, err)

		var ev map[string]interface{}
		err = json.Unmarshal(content, &ev)
		require.NoError(t, err)

		expected = append(expected, ev)
	}
	return expected
}

// TestMapperWithMetricSetAndInfo tests mapping methods with Info fields
func TestMapperWithMetricSetAndInfo(t *testing.T, glob string, ms MetricSetAPI, mapper func(mb.ReporterV2, MetricSetAPI, Info, []byte, bool) error) {
	files, err := filepath.Glob(glob)
	require.NoError(t, err)
	// Makes sure glob matches at least 1 file
	require.True(t, len(files) > 0)

	info := Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			err = mapper(reporter, ms, info, input, true)
			require.NoError(t, err)
			require.True(t, len(reporter.GetEvents()) >= 1)
			require.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}

// TestMapperWithMetricSetAndInfo tests mapping methods with Info fields
func TestMapperWithHttpHelper(t *testing.T, glob string, httpClient *helper.HTTP,
	mapper func(mb.ReporterV2, *helper.HTTP, Info, []byte, bool, *logp.Logger) error) {
	files, err := filepath.Glob(glob)
	require.NoError(t, err)
	// Makes sure glob matches at least 1 file
	require.True(t, len(files) > 0)

	info := Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
		Version: Version{Number: &version.V{
			Major:  7,
			Minor:  6,
			Bugfix: 0,
		}},
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			err = mapper(reporter, httpClient, info, input, true, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err)
			require.True(t, len(reporter.GetEvents()) >= 1)
			require.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}
