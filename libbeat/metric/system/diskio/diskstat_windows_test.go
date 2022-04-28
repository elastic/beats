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

//go:build !integration && windows
// +build !integration,windows

package diskio

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCDriveFilterOnWindowsTestEnv(t *testing.T) {
	// Test if flakey.
	t.Skip("See https://github.com/elastic/beats/issues/17126")

	conf := map[string]interface{}{
		"module":                 "system",
		"metricsets":             []string{"diskio"},
		"diskio.include_devices": []string{"C:"},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, conf)
	data, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs)
	assert.Equal(t, 1, len(data))
	assert.Equal(t, data[0].MetricSetFields["name"], "C:")
	reads := data[0].MetricSetFields["read"].(mapstr.M)
	writes := data[0].MetricSetFields["write"].(mapstr.M)
	// Check values
	readCount := reads["count"].(uint64)
	readBytes := reads["bytes"].(uint64)
	readTime := reads["time"].(uint64)
	writeCount := writes["count"].(uint64)
	writeBytes := writes["bytes"].(uint64)
	writeTime := writes["time"].(uint64)

	assert.True(t, readCount > 0)
	assert.True(t, readBytes > 0)
	assert.True(t, readTime > 0)

	assert.True(t, writeCount > 0)
	assert.True(t, writeBytes > 0)
	assert.True(t, writeTime > 0)
	err := disablePerformanceCounters(`\\.\C:`)
	assert.NoError(t, err)
}

func TestAllDrivesOnWindowsTestEnv(t *testing.T) {
	conf := map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"diskio"},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, conf)
	data, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs)
	assert.True(t, len(data) >= 1)
	drives, err := getLogicalDriveStrings()
	assert.NoError(t, err)
	for _, drive := range drives {
		err := disablePerformanceCounters(drive.UNCPath)
		assert.NoError(t, err)
	}
}
