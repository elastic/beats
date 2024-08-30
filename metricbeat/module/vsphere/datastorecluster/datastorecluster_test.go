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

package datastorecluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/simulator"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.VPX()
	model.Pod = 1
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

	name, ok := event["name"].(string)
	require.True(t, ok, "Expected 'name' field to be of type mapstr.M")
	assert.NotNil(t, name, "Expected 'name' field to be non-nil")

	capacity, ok := event["capacity"].(mapstr.M)
	require.True(t, ok, "Expected 'capacity' field to be of type mapstr.M")
	assert.GreaterOrEqual(t, capacity["bytes"], int64(0), "Expected 'capacity.bytes' to be non-negative")

	freeSpace, ok := event["free_space"].(mapstr.M)
	require.True(t, ok, "Expected 'free_space' field to be of type mapstr.M")
	assert.GreaterOrEqual(t, freeSpace["bytes"], int64(0), "Expected 'free_space.bytes' to be non-negative")
}

func TestDatastoreCluster(t *testing.T) {
	model := simulator.VPX()
	model.Pod = 1
	err := model.Create()
	require.NoError(t, err, "failed to create model")
	t.Cleanup(func() { model.Remove() })

	ts := model.Service.NewServer()
	t.Cleanup(func() { ts.Close() })

	f := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(ts))

	err = mbtest.WriteEventsReporterV2WithContext(f, t, "")
	assert.NoError(t, err, "failed to write events with reporter")
}

func getConfig(ts *simulator.Server) map[string]interface{} {
	return map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"datastorecluster"},
		"hosts":      []string{ts.URL.String()},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
