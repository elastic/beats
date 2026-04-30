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

package datastore

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	t.Skip("Flaky test: https://github.com/elastic/beats/issues/43252")
	// Creating a new simulator model with VPX server to collect broad range of data.
	model := simulator.VPX()
	err := model.Create()
	require.NoError(t, err, "failed to create model")
	t.Cleanup(model.Remove)

	ts := model.Service.NewServer()
	t.Cleanup(ts.Close)

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))
	events, errs := mbtest.ReportingFetchV2WithContext(f)
	require.Empty(t, errs, "Expected no errors during fetch")
	require.NotEmpty(t, events, "Expected to receive at least one event")

	event := events[0].MetricSetFields

	t.Logf("Fetched event from %s/%s event: %+v", f.Module().Name(), f.Name(), event)

	assert.EqualValues(t, "LocalDS_0", event["name"])
	assert.EqualValues(t, "OTHER", event["fstype"])

	// Values are based on the result 'df -k'.
	fields := []string{
		"capacity.total.bytes",
		"capacity.free.bytes",
		"status",
		"host.count",
		"vm.count",
		"write.bytes",
		"capacity.used.bytes",
		"disk.capacity.usage.bytes",
		"disk.capacity.bytes",
		"disk.provisioned.bytes",
	}
	for _, field := range fields {
		value, err := event.GetValue(field)
		if err != nil {
			t.Error(field, err)
			return
		}
		switch field {
		case "status":
			assert.NotNil(t, value)
		case "vm.count", "host.count":
			assert.GreaterOrEqual(t, value, 0)
		default:
			assert.GreaterOrEqual(t, value, int64(0))
		}
	}
}

func TestDataStoreMetricSetData(t *testing.T) {
	model := simulator.ESX()
	err := model.Create()
	require.NoError(t, err, "failed to create model")
	t.Cleanup(model.Remove)

	ts := model.Service.NewServer()
	t.Cleanup(ts.Close)

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))

	err = mbtest.WriteEventsReporterV2WithContext(f, t, "")
	assert.NoError(t, err, "failed to write events with reporter")
}

func getConfig(ts *simulator.Server) map[string]interface{} {
	return map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"datastore"},
		"hosts":      []string{ts.URL.String()},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
